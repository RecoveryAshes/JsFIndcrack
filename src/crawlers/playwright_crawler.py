#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Playwright动态爬虫模块 - 使用内置浏览器引擎
支持Chromium、Firefox和WebKit引擎，无需外部浏览器依赖
"""

import asyncio
import time
import re
import sys
import os
from pathlib import Path
from typing import Set, Dict, List, Optional, Tuple
from urllib.parse import urljoin, urlparse
import logging
from tqdm.asyncio import tqdm

try:
    from playwright.async_api import async_playwright, Browser, Page, BrowserContext
    PLAYWRIGHT_AVAILABLE = True
except ImportError:
    PLAYWRIGHT_AVAILABLE = False
    # 为类型注解提供占位符
    Browser = None
    Page = None
    BrowserContext = None

from ..utils.utils import is_supported_file, generate_file_path, convert_to_utf8, format_file_size, get_content_hash, is_duplicate_content, is_file_already_downloaded
from ..utils.logger import get_logger


def get_packaged_browser_path():
    """获取打包环境中的浏览器路径"""
    if getattr(sys, 'frozen', False):
        # 在打包环境中
        if sys.platform == "darwin":  # macOS
            # 在打包的可执行文件中，浏览器位于相对路径
            base_path = Path(sys._MEIPASS)
            browser_path = base_path / "playwright_browsers" / "chromium-1187" / "chrome-mac" / "Chromium.app" / "Contents" / "MacOS" / "Chromium"
            if browser_path.exists():
                return str(browser_path)
        elif sys.platform.startswith("linux"):
            # Linux环境
            base_path = Path(sys._MEIPASS)
            browser_path = base_path / "playwright_browsers" / "chromium-1187" / "chrome-linux" / "chrome"
            if browser_path.exists():
                return str(browser_path)
        elif sys.platform.startswith("win"):
            # Windows环境
            base_path = Path(sys._MEIPASS)
            browser_path = base_path / "playwright_browsers" / "chromium-1187" / "chrome-win" / "chrome.exe"
            if browser_path.exists():
                return str(browser_path)
    return None


class PlaywrightCrawler:
    """使用Playwright的动态JavaScript和Source Map爬虫"""
    
    def __init__(self, target_url: str = None, max_depth: int = 2, wait_time: int = 3, 
                 max_workers: int = 4, browser_type: str = "chromium", headless: bool = True,
                 existing_file_hashes: Dict[str, str] = None):
        """
        初始化Playwright爬虫
        
        Args:
            target_url: 目标URL
            max_depth: 最大爬取深度
            wait_time: 页面等待时间（秒）
            max_workers: 最大并发数（控制同时打开的标签页数量）
            browser_type: 浏览器类型 ("chromium", "firefox", "webkit")
            headless: 是否无头模式运行
            existing_file_hashes: 已有文件的哈希值字典，用于跨模式去重
        """
        if not PLAYWRIGHT_AVAILABLE:
            raise ImportError("Playwright未安装，请运行: pip install playwright && playwright install")
        
        self.target_url = target_url
        self.max_depth = max_depth
        self.wait_time = wait_time
        self.max_workers = max_workers
        self.browser_type = browser_type.lower()
        self.headless = headless
        
        # 验证浏览器类型
        if self.browser_type not in ["chromium", "firefox", "webkit"]:
            self.browser_type = "chromium"
        
        self.logger = get_logger("playwright_crawler")
        
        # 统计信息
        self.stats = {
            'pages_visited': 0,
            'js_files_found': 0,
            'js_files_downloaded': 0,
            'js_files_failed': 0,
            'js_files_duplicated': 0,  # 新增：重复文件数量
            'js_files_cross_mode_duplicated': 0,  # 新增：跨模式重复文件数量
            'total_size': 0,
            'start_time': None,
            'end_time': None
        }
        
        # 去重相关
        self.content_hashes = set()  # 存储已下载文件的内容哈希
        self.hash_to_filename = {}   # 哈希值到文件名的映射
        self.download_tasks = []  # 存储下载任务
        
        # 初始化已有文件哈希（用于跨模式去重）
        self.static_file_hashes = set()  # 存储来自静态爬取的文件哈希
        if existing_file_hashes:
            self.content_hashes.update(existing_file_hashes.keys())
            self.hash_to_filename.update(existing_file_hashes)
            self.static_file_hashes.update(existing_file_hashes.keys())
            self.logger.info(f"加载了 {len(existing_file_hashes)} 个已有文件哈希，用于跨模式去重")
        
        # 存储已访问的URL和发现的JS文件
        self.visited_urls: Set[str] = set()
        self.discovered_js_files: Set[str] = set()
        
        # 标签页管理
        self.active_pages = {}  # {page_id: page_object}
        self.page_semaphore = None  # 将在__aenter__中初始化
        self.download_queue = None  # 将在__aenter__中初始化
        self.completed_downloads = set()  # 已完成下载的URL
        
        # Playwright对象
        self.playwright = None
        self.browser: Optional[Browser] = None
        self.context: Optional[BrowserContext] = None

    async def __aenter__(self):
        """异步上下文管理器入口"""
        # 初始化异步对象
        self.page_semaphore = asyncio.Semaphore(self.max_workers)
        self.download_queue = asyncio.Queue()
        
        await self._setup_browser()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """异步上下文管理器出口"""
        await self._cleanup_browser()

    async def _setup_browser(self):
        """设置浏览器"""
        try:
            self.playwright = await async_playwright().start()
            
            # 检查是否在打包环境中，如果是则使用打包的浏览器
            packaged_browser_path = get_packaged_browser_path()
            
            # 浏览器启动参数
            browser_args = [
                '--no-sandbox',
                '--disable-dev-shm-usage',
                '--disable-gpu',
                '--disable-web-security',
                '--disable-features=VizDisplayCompositor',
                '--disable-extensions',
                '--disable-plugins',
                '--disable-images',
                '--disable-javascript-harmony-shipping',
                '--disable-background-timer-throttling',
                '--disable-backgrounding-occluded-windows',
                '--disable-renderer-backgrounding'
            ]
            
            # 根据类型选择浏览器
            if self.browser_type == "firefox":
                launch_options = {
                    'headless': self.headless,
                    'args': ['--no-sandbox', '--disable-dev-shm-usage']
                }
                if packaged_browser_path and 'firefox' in packaged_browser_path.lower():
                    launch_options['executable_path'] = packaged_browser_path
                self.browser = await self.playwright.firefox.launch(**launch_options)
                
            elif self.browser_type == "webkit":
                launch_options = {
                    'headless': self.headless
                }
                if packaged_browser_path and 'webkit' in packaged_browser_path.lower():
                    launch_options['executable_path'] = packaged_browser_path
                self.browser = await self.playwright.webkit.launch(**launch_options)
                
            else:  # chromium (默认)
                launch_options = {
                    'headless': self.headless,
                    'args': browser_args
                }
                if packaged_browser_path:
                    launch_options['executable_path'] = packaged_browser_path
                    self.logger.info(f"使用打包的浏览器: {packaged_browser_path}")
                
                self.browser = await self.playwright.chromium.launch(**launch_options)
            
            # 创建浏览器上下文
            self.context = await self.browser.new_context(
                viewport={'width': 1920, 'height': 1080},
                user_agent='Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
                ignore_https_errors=True,
                java_script_enabled=True
            )
            
            self.logger.info(f"{self.browser_type.title()}浏览器初始化成功")
            
        except Exception as e:
            self.logger.error(f"浏览器初始化失败: {e}")
            raise

    async def _cleanup_browser(self):
        """清理浏览器资源"""
        try:
            # 关闭所有活跃的页面
            for page_id, page in self.active_pages.items():
                try:
                    await page.close()
                except:
                    pass
            self.active_pages.clear()
            
            if self.context:
                await self.context.close()
            if self.browser:
                await self.browser.close()
            if self.playwright:
                await self.playwright.stop()
            self.logger.info("🧹 浏览器资源已清理")
        except Exception as e:
            self.logger.error(f"清理浏览器资源时出错: {e}")

    async def _download_js_with_tab_control(self, url: str, output_dir: Path) -> Optional[Dict]:
        """使用标签页控制下载JavaScript文件"""
        if url in self.completed_downloads:
            return None
        
        # 检查文件是否已存在（基于文件名）
        if is_file_already_downloaded(url, self.target_url, 'dynamic'):
            parsed_url = urlparse(url)
            filename = Path(parsed_url.path).name or 'script.js'
            self.logger.info(f"跳过已存在文件: {filename}")
            self.completed_downloads.add(url)
            return None
            
        async with self.page_semaphore:  # 控制同时打开的标签页数量
            page_id = f"download_{len(self.active_pages)}"
            page = None
            
            try:
                # 创建新页面
                page = await self.context.new_page()
                self.active_pages[page_id] = page
                
                # 显示下载开始信息
                parsed_url = urlparse(url)
                filename = Path(parsed_url.path).name or 'script.js'
                self.logger.info(f"📥 开始下载 [{self.stats['js_files_downloaded'] + 1}] {filename}")
                self.logger.info(f"URL: {url}")
                self.logger.info(f" 标签页状态: {len(self.active_pages)}/{self.max_workers} 活跃")
                
                # 直接访问JS文件URL
                response = await page.goto(url, timeout=10000)
                
                if response and response.status == 200:
                    content = await response.text()
                    content_bytes = content.encode('utf-8')
                    content_size = len(content_bytes)
                    
                    # 检查内容是否重复
                    content_hash = get_content_hash(content_bytes)
                    if content_hash in self.content_hashes:
                        existing_filename = self.hash_to_filename.get(content_hash, "未知文件")
                        
                        # 判断是跨模式去重还是模式内去重
                        is_cross_mode = content_hash in self.static_file_hashes
                        
                        if is_cross_mode:
                            self.logger.info(f"跳过跨模式重复文件: {filename} (与静态爬取的 {existing_filename} 内容相同)")
                            self.stats['js_files_cross_mode_duplicated'] += 1
                        else:
                            self.logger.info(f"跳过重复文件: {filename} (与 {existing_filename} 内容相同)")
                            self.stats['js_files_duplicated'] += 1
                        
                        self.completed_downloads.add(url)
                        return {
                            'url': url,
                            'path': None,  # 未保存新文件
                            'size': content_size,
                            'duplicate': True,
                            'cross_mode_duplicate': is_cross_mode,
                            'original_file': existing_filename
                        }
                    
                    # 生成文件路径
                    if not filename.endswith('.js'):
                        filename += '.js'
                    
                    # 避免文件名冲突
                    counter = 1
                    original_filename = filename
                    while (output_dir / filename).exists():
                        name, ext = original_filename.rsplit('.', 1)
                        filename = f"{name}_{counter}.{ext}"
                        counter += 1
                    
                    file_path = output_dir / filename
                    
                    # 保存文件
                    with open(file_path, 'w', encoding='utf-8') as f:
                        f.write(content)
                    
                    # 更新去重信息
                    self.content_hashes.add(content_hash)
                    self.hash_to_filename[content_hash] = filename
                    
                    self.completed_downloads.add(url)
                    self.stats['js_files_downloaded'] += 1
                    self.stats['total_size'] += content_size
                    
                    # 格式化文件大小
                    if content_size < 1024:
                        size_str = f"{content_size} B"
                    elif content_size < 1024 * 1024:
                        size_str = f"{content_size / 1024:.1f} KB"
                    else:
                        size_str = f"{content_size / (1024 * 1024):.1f} MB"
                    
                    self.logger.info(f"下载完成: {filename} ({size_str})")
                    self.logger.info(f"保存路径: {file_path}")
                    
                    return {
                        'url': url,
                        'path': str(file_path),
                        'size': content_size,
                        'filename': filename,
                        'duplicate': False,
                        'content_hash': content_hash
                    }
                else:
                    self.logger.warning(f"下载失败: {url} (状态码: {response.status if response else 'None'})")
                    self.stats['js_files_failed'] += 1
                    
            except Exception as e:
                self.logger.error(f"下载JS文件失败 {url}: {e}")
                self.stats['js_files_failed'] += 1
            finally:
                # 立即关闭页面释放资源
                if page:
                    try:
                        await page.close()
                        if page_id in self.active_pages:
                            del self.active_pages[page_id]
                        self.logger.debug(f"🗑️ 已关闭标签页: {page_id} (剩余活跃标签页: {len(self.active_pages)})")
                    except:
                        pass
        
        return None

    async def _start_controlled_downloads(self, js_files: Set[str], output_dir: Path):
        """启动受控的下载任务"""
        new_files = js_files - self.completed_downloads
        if not new_files:
            return
            
        self.logger.info(f"🚀 发现 {len(new_files)} 个新的JS文件，开始下载...")
        
        # 创建下载进度条
        download_progress = tqdm(
            total=len(new_files),
            desc="📥 下载JS文件",
            unit="文件",
            dynamic_ncols=True,
            colour="green",
            position=0,
            leave=False,
            ncols=100,
            bar_format='{desc}: {percentage:3.0f}%|{bar}| {n}/{total} [{elapsed}<{remaining}, {postfix}]'
        )
        
        # 创建下载任务
        download_tasks = []
        for url in new_files:
            task = asyncio.create_task(self._download_js_with_progress(url, output_dir, download_progress))
            download_tasks.append(task)
        
        # 等待所有下载完成
        if download_tasks:
            await asyncio.gather(*download_tasks, return_exceptions=True)
        
        download_progress.close()

    async def _download_js_with_progress(self, url: str, output_dir: Path, progress_bar: tqdm) -> Optional[Dict]:
        """带进度条的JS文件下载"""
        try:
            result = await self._download_js_with_tab_control(url, output_dir)
            
            # 更新进度条
            if result:
                file_size = result.get('file_size', 0)
                progress_bar.set_postfix({
                    "已完成": len(self.completed_downloads),
                    "最新": result.get('filename', '')[:20] + ('...' if len(result.get('filename', '')) > 20 else '')
                })
            
            progress_bar.update(1)
            return result
            
        except Exception as e:
            self.logger.error(f"下载失败 {url}: {e}")
            progress_bar.update(1)
            return None

    async def _extract_files_from_page(self, page: Page, base_url: str) -> Set[str]:
        """从页面中提取JavaScript和Source Map文件URL"""
        file_urls = set()
        
        try:
            # 等待页面基本加载完成（不等待networkidle，加载过程中也能收集JS）
            await page.wait_for_load_state('domcontentloaded', timeout=10000)
            
            # 执行JavaScript来获取所有script标签
            script_urls = await page.evaluate("""
                () => {
                    const scripts = Array.from(document.querySelectorAll('script[src]'));
                    return scripts.map(script => script.src).filter(src => src);
                }
            """)
            
            # 处理相对URL
            for url in script_urls:
                if url:
                    absolute_url = urljoin(base_url, url)
                    if self._is_supported_file(absolute_url):
                        file_urls.add(absolute_url)
            
            # 监听网络请求中的JS和MAP文件
            network_files = await page.evaluate("""
                () => {
                    return window.jsFiles || [];
                }
            """)
            
            for url in network_files:
                if url:
                    absolute_url = urljoin(base_url, url)
                    if self._is_supported_file(absolute_url):
                        file_urls.add(absolute_url)
                        
        except Exception as e:
            self.logger.warning(f"提取文件时出错: {e}")
        
        return file_urls

    def _is_supported_file(self, url: str) -> bool:
        """判断URL是否为支持的文件类型（JavaScript或Source Map）"""
        if not url:
            return False
        
        # 使用统一的文件类型检测函数
        return is_supported_file(url)

    async def _start_immediate_downloads(self, js_files: Set[str]):
        """立即开始下载新发现的JS文件"""
        for js_url in js_files:
            if js_url not in self.discovered_js_files:
                self.discovered_js_files.add(js_url)
                # 创建下载任务但不等待完成
                task = asyncio.create_task(self._download_file_immediate(js_url))
                self.download_tasks.append(task)
                self.logger.debug(f"已启动下载任务: {js_url}")

    async def _download_file_immediate(self, url: str) -> Optional[Dict]:
        """立即下载文件（用于并发下载）"""
        try:
            # 创建新页面用于下载
            page = await self.context.new_page()
            
            # 直接访问JS文件URL
            response = await page.goto(url, timeout=10000)
            
            if response and response.status == 200:
                content = await response.text()
                
                # 生成文件路径
                file_path = generate_file_path(url, self.target_url, 'dynamic')
                
                # 保存文件
                with open(file_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                
                # 转换编码为UTF-8
                convert_to_utf8(file_path)
                
                self.logger.info(f"下载成功: {url} -> {file_path}")
                
                await page.close()
                
                # 更新统计信息
                self.stats['js_files_downloaded'] += 1
                self.stats['total_size'] += len(content.encode('utf-8'))
                
                return {
                    'url': url,
                    'path': str(file_path),
                    'size': len(content.encode('utf-8'))
                }
            else:
                self.logger.warning(f"下载失败: {url} (状态码: {response.status if response else 'None'})")
                self.stats['js_files_failed'] += 1
                
            await page.close()
            
        except Exception as e:
            self.logger.error(f"下载JS文件失败 {url}: {e}")
            self.stats['js_files_failed'] += 1
        
        return None

    async def _visit_page(self, url: str, depth: int = 0, output_dir: Path = None) -> Tuple[Set[str], Set[str]]:
        """访问单个页面并提取JS文件和链接"""
        if depth > self.max_depth or url in self.visited_urls:
            return set(), set()
        
        self.visited_urls.add(url)
        js_files = set()
        links = set()
        
        try:
            page = await self.context.new_page()
            
            # 设置网络监听来捕获JS文件请求
            js_requests = []
            
            async def handle_request(request):
                if self._is_supported_file(request.url):
                    js_requests.append(request.url)
                    self.logger.debug(f"🔍 发现文件请求: {request.url}")
            
            page.on('request', handle_request)
            
            # 显示详细的爬取信息
            depth_indicator = "  " * depth + "└─" if depth > 0 else ""
            self.logger.info(f"🌐 {depth_indicator}正在爬取 [深度 {depth}]: {url}")
            self.logger.info(f" 爬取进度: 已访问 {len(self.visited_urls)} 个页面")
            
            # 访问页面（不等待networkidle，这样可以在加载过程中收集JS文件）
            response = await page.goto(url, wait_until='domcontentloaded', timeout=10000)
            
            if response and response.status >= 400:
                self.logger.warning(f"⚠️ 页面返回错误状态: {response.status}")
            
            # 短暂等待让初始JS加载（减少等待时间）
            await asyncio.sleep(1)
            
            # 触发可能的动态加载
            await page.evaluate("""
                () => {
                    // 滚动页面触发懒加载
                    window.scrollTo(0, document.body.scrollHeight);
                    
                    // 触发常见的事件
                    ['click', 'mouseover', 'focus'].forEach(eventType => {
                        document.querySelectorAll('button, a, input').forEach(el => {
                            try {
                                el.dispatchEvent(new Event(eventType, {bubbles: true}));
                            } catch(e) {}
                        });
                    });
                }
            """)
            
            # 再次短暂等待
            await asyncio.sleep(1)
            
            # 提取JS和MAP文件
            page_files = await self._extract_files_from_page(page, url)
            js_files.update(page_files)
            js_files.update(js_requests)
            
            # 立即开始下载新发现的文件（使用标签页控制）
            if js_files and output_dir:
                await self._start_controlled_downloads(js_files, output_dir)
            
            # 提取页面链接（用于深度爬取）
            if depth < self.max_depth:
                try:
                    page_links = await page.evaluate("""
                        () => {
                            const links = Array.from(document.querySelectorAll('a[href]'));
                            return links.map(link => link.href).filter(href => href);
                        }
                    """)
                    
                    base_domain = urlparse(url).netloc
                    for link in page_links:
                        if link and urlparse(link).netloc == base_domain:
                            links.add(link)
                            
                except Exception as e:
                    self.logger.warning(f"提取链接时出错: {e}")
            
            await page.close()
            self.stats['pages_visited'] += 1
            
            # 显示页面爬取结果
            depth_indicator = "  " * depth + "└─" if depth > 0 else ""
            if js_files:
                self.logger.info(f" {depth_indicator}页面分析完成: 发现 {len(js_files)} 个JS文件")
                for js_file in list(js_files)[:3]:  # 只显示前3个，避免日志过长
                    self.logger.debug(f"  {Path(urlparse(js_file).path).name}")
                if len(js_files) > 3:
                    self.logger.debug(f"   ... 还有 {len(js_files) - 3} 个文件")
            else:
                self.logger.info(f" {depth_indicator}页面分析完成: 未发现JS文件")
            
            if links:
                self.logger.info(f"发现 {len(links)} 个内部链接用于深度爬取")
            
        except Exception as e:
            self.logger.error(f"访问页面失败 {url}: {e}")
        
        return js_files, links

    async def crawl_website(self, start_url: str, output_dir: Path) -> Dict:
        """爬取网站的JavaScript文件"""
        self.logger.info(f"🚀 开始动态爬取")
        self.logger.info(f" 目标网站: {start_url}")
        self.logger.info(f" 爬取配置: 最大深度 {self.max_depth}, 标签页控制 {self.max_workers}")
        self.logger.info(f"🌐 浏览器模式: {'无头模式' if self.headless else '有头模式'} ({self.browser_type})")
        self.logger.info(f"📁 输出目录: {output_dir}")
        self.logger.info(f"{'='*60}")
        
        try:
            # 确保输出目录存在
            output_dir.mkdir(parents=True, exist_ok=True)
            self.stats['start_time'] = time.time()
            
            # 开始爬取
            to_visit = [(start_url, 0)]
            all_js_files = set()
            
            # 估算总页面数（基于深度）
            estimated_pages = min(100, sum(10 ** i for i in range(self.max_depth + 1)))
            
            # 创建页面访问进度条
            page_progress = tqdm(
                total=estimated_pages,
                desc="🌐 页面爬取",
                unit="页面",
                dynamic_ncols=True,
                colour="blue",
                position=0,
                leave=False,
                ncols=100,
                bar_format='{desc}: {percentage:3.0f}%|{bar}| {n}/{total} [{elapsed}<{remaining}, {postfix}]'
            )
            
            while to_visit:
                current_url, depth = to_visit.pop(0)
                
                if depth > self.max_depth:
                    continue
                
                # 更新进度条描述
                page_progress.set_description(f"🌐 爬取深度{depth}")
                
                js_files, links = await self._visit_page(current_url, depth, output_dir)
                all_js_files.update(js_files)
                
                # 更新进度条
                page_progress.update(1)
                page_progress.set_postfix({
                    "已访问": len(self.visited_urls),
                    "待访问": len(to_visit),
                    "JS文件": len(all_js_files)
                })
                
                # 添加新发现的链接到待访问列表
                for link in links:
                    if link not in self.visited_urls:
                        to_visit.append((link, depth + 1))
            
            page_progress.close()
            
            self.stats['js_files_found'] = len(all_js_files)
            self.stats['total_discovered'] = len(all_js_files)  # 添加total_discovered字段
            self.stats['end_time'] = time.time()
            
            # 下载剩余的JS文件
            remaining_files = all_js_files - self.completed_downloads
            if remaining_files:
                self.logger.info(f"📥 下载剩余的 {len(remaining_files)} 个JavaScript文件...")
                await self._start_controlled_downloads(remaining_files, output_dir)
            
            duration = self.stats['end_time'] - self.stats['start_time']
            
            # 详细的统计信息
            total_found = len(all_js_files)
            downloaded = self.stats['js_files_downloaded']
            duplicated = self.stats['js_files_duplicated']
            cross_mode_duplicated = self.stats['js_files_cross_mode_duplicated']
            failed = self.stats['js_files_failed']
            total_duplicated = duplicated + cross_mode_duplicated
            
            self.logger.info(f"爬取完成! 发现 {total_found} 个JS文件，下载 {downloaded} 个，重复 {total_duplicated} 个，失败 {failed} 个，耗时 {duration:.2f}秒")
            
            if total_duplicated > 0:
                if cross_mode_duplicated > 0:
                    self.logger.info(f"去重效果: 节省了 {total_duplicated} 个重复文件的下载 (其中 {cross_mode_duplicated} 个为跨模式去重)")
                else:
                    self.logger.info(f"去重效果: 节省了 {total_duplicated} 个重复文件的下载")
            
        except Exception as e:
            self.logger.error(f"动态爬取失败: {e}")
            raise
        
        # 添加去重统计信息到返回结果
        result = self.stats.copy()
        result['duplicated_files'] = self.stats['js_files_duplicated']
        result['cross_mode_duplicated_files'] = self.stats['js_files_cross_mode_duplicated']
        return result




async def main():
    """测试函数"""
    import sys
    
    if len(sys.argv) < 2:
        print("用法: python playwright_crawler.py <URL>")
        return
    
    url = sys.argv[1]
    output_dir = Path("test_output")
    
    async with PlaywrightCrawler(max_depth=1, wait_time=3) as crawler:
        stats = await crawler.crawl_website(url, output_dir)
        print(f"爬取完成: {stats}")


if __name__ == "__main__":
    asyncio.run(main())