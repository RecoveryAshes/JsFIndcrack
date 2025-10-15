#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Playwright动态爬虫模块 - 使用内置浏览器引擎
支持Chromium、Firefox和WebKit引擎，无需外部浏览器依赖
"""

import asyncio
import time
import re
from pathlib import Path
from typing import Set, Dict, List, Optional, Tuple
from urllib.parse import urljoin, urlparse
import logging

try:
    from playwright.async_api import async_playwright, Browser, Page, BrowserContext
    PLAYWRIGHT_AVAILABLE = True
except ImportError:
    PLAYWRIGHT_AVAILABLE = False

from ..utils.utils import is_javascript_file, generate_file_path, convert_to_utf8, format_file_size


class PlaywrightCrawler:
    """使用Playwright的动态JavaScript爬虫"""
    
    def __init__(self, target_url: str = None, max_depth: int = 2, wait_time: int = 3, 
                 max_workers: int = 4, browser_type: str = "chromium"):
        """
        初始化Playwright爬虫
        
        Args:
            target_url: 目标URL
            max_depth: 最大爬取深度
            wait_time: 页面等待时间（秒）
            max_workers: 最大并发数
            browser_type: 浏览器类型 ("chromium", "firefox", "webkit")
        """
        if not PLAYWRIGHT_AVAILABLE:
            raise ImportError("Playwright未安装，请运行: pip install playwright && playwright install")
        
        self.target_url = target_url
        self.max_depth = max_depth
        self.wait_time = wait_time
        self.max_workers = max_workers
        self.browser_type = browser_type.lower()
        
        # 验证浏览器类型
        if self.browser_type not in ["chromium", "firefox", "webkit"]:
            self.browser_type = "chromium"
        
        self.logger = logging.getLogger(__name__)
        
        # 统计信息
        self.stats = {
            'pages_visited': 0,
            'js_files_found': 0,
            'js_files_downloaded': 0,
            'js_files_failed': 0,
            'total_size': 0
        }
        self.download_tasks = []  # 存储下载任务
        
        # 存储已访问的URL和发现的JS文件
        self.visited_urls: Set[str] = set()
        self.discovered_js_files: Set[str] = set()
        
        # Playwright对象
        self.playwright = None
        self.browser: Optional[Browser] = None
        self.context: Optional[BrowserContext] = None

    async def __aenter__(self):
        """异步上下文管理器入口"""
        await self._setup_browser()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """异步上下文管理器出口"""
        await self._cleanup_browser()

    async def _setup_browser(self):
        """设置浏览器"""
        try:
            self.playwright = await async_playwright().start()
            
            # 根据类型选择浏览器
            if self.browser_type == "firefox":
                self.browser = await self.playwright.firefox.launch(
                    headless=False,
                    args=['--no-sandbox', '--disable-dev-shm-usage']
                )
            elif self.browser_type == "webkit":
                self.browser = await self.playwright.webkit.launch(
                    headless=False
                )
            else:  # chromium (默认)
                self.browser = await self.playwright.chromium.launch(
                    headless=False,
                    args=[
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
                )
            
            # 创建浏览器上下文
            self.context = await self.browser.new_context(
                viewport={'width': 1920, 'height': 1080},
                user_agent='Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
                ignore_https_errors=True,
                java_script_enabled=True
            )
            
            self.logger.info(f"✅ {self.browser_type.title()}浏览器初始化成功")
            
        except Exception as e:
            self.logger.error(f"❌ 浏览器初始化失败: {e}")
            raise

    async def _cleanup_browser(self):
        """清理浏览器资源"""
        try:
            if self.context:
                await self.context.close()
            if self.browser:
                await self.browser.close()
            if self.playwright:
                await self.playwright.stop()
            self.logger.info("🧹 浏览器资源已清理")
        except Exception as e:
            self.logger.error(f"清理浏览器资源时出错: {e}")

    async def _extract_js_files_from_page(self, page: Page, base_url: str) -> Set[str]:
        """从页面中提取JavaScript文件URL"""
        js_files = set()
        
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
                    if self._is_js_file(absolute_url):
                        js_files.add(absolute_url)
            
            # 监听网络请求中的JS文件
            network_js_files = await page.evaluate("""
                () => {
                    return window.jsFiles || [];
                }
            """)
            
            for url in network_js_files:
                if url:
                    absolute_url = urljoin(base_url, url)
                    if self._is_js_file(absolute_url):
                        js_files.add(absolute_url)
                        
        except Exception as e:
            self.logger.warning(f"提取JS文件时出错: {e}")
        
        return js_files

    def _is_js_file(self, url: str) -> bool:
        """判断URL是否为JavaScript文件"""
        if not url:
            return False
        
        # 移除查询参数和片段
        clean_url = url.split('?')[0].split('#')[0]
        
        # 检查文件扩展名
        if clean_url.endswith('.js'):
            return True
        
        # 检查MIME类型相关的URL模式
        js_patterns = [
            r'\.js$',
            r'\.js\?',
            r'/js/',
            r'javascript',
            r'\.min\.js',
            r'application/javascript',
            r'text/javascript'
        ]
        
        return any(re.search(pattern, url, re.IGNORECASE) for pattern in js_patterns)

    async def _start_immediate_downloads(self, js_files: Set[str]):
        """立即开始下载新发现的JS文件"""
        for js_url in js_files:
            if js_url not in self.discovered_js_files:
                self.discovered_js_files.add(js_url)
                # 创建下载任务但不等待完成
                task = asyncio.create_task(self._download_js_file_immediate(js_url))
                self.download_tasks.append(task)
                self.logger.debug(f"已启动下载任务: {js_url}")

    async def _download_js_file_immediate(self, url: str) -> Optional[Dict]:
        """立即下载JS文件（用于并发下载）"""
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

    async def _visit_page(self, url: str, depth: int = 0) -> Tuple[Set[str], Set[str]]:
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
                if self._is_js_file(request.url):
                    js_requests.append(request.url)
            
            page.on('request', handle_request)
            
            self.logger.info(f"正在访问页面: {url}")
            
            # 访问页面（不等待networkidle，这样可以在加载过程中收集JS文件）
            response = await page.goto(url, wait_until='domcontentloaded', timeout=10000)
            
            if response and response.status >= 400:
                self.logger.warning(f"页面返回错误状态: {response.status}")
            
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
            
            # 提取JS文件
            page_js_files = await self._extract_js_files_from_page(page, url)
            js_files.update(page_js_files)
            js_files.update(js_requests)
            
            # 立即开始下载新发现的JS文件
            await self._start_immediate_downloads(js_files)
            
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
            
            if js_files:
                self.logger.info(f"在页面 {url} 中发现 {len(js_files)} 个JavaScript文件")
            
        except Exception as e:
            self.logger.error(f"访问页面失败 {url}: {e}")
        
        return js_files, links

    async def crawl_website(self, start_url: str, output_dir: Path) -> Dict:
        """爬取网站的JavaScript文件"""
        self.logger.info(f"开始动态爬取: {start_url}")
        
        try:
            # 确保输出目录存在
            output_dir.mkdir(parents=True, exist_ok=True)
            
            # 开始爬取
            to_visit = [(start_url, 0)]
            all_js_files = set()
            
            while to_visit:
                current_url, depth = to_visit.pop(0)
                
                if depth > self.max_depth:
                    continue
                
                js_files, links = await self._visit_page(current_url, depth)
                all_js_files.update(js_files)
                
                # 添加新发现的链接到待访问列表
                for link in links:
                    if link not in self.visited_urls:
                        to_visit.append((link, depth + 1))
            
            self.stats['js_files_found'] = len(all_js_files)
            self.logger.info(f"发现 {len(all_js_files)} 个JavaScript文件")
            
            # 等待所有并发下载任务完成
            if self.download_tasks:
                self.logger.info(f"等待 {len(self.download_tasks)} 个下载任务完成...")
                results = await asyncio.gather(*self.download_tasks, return_exceptions=True)
                
                for result in results:
                    if isinstance(result, Exception):
                        self.stats['js_files_failed'] += 1
                    elif result:
                        self.stats['js_files_downloaded'] += 1
                        self.stats['total_size'] += result.get('size', 0)
            
        except Exception as e:
            self.logger.error(f"动态爬取失败: {e}")
            raise
        
        return self.stats

    async def _download_js_file(self, url: str, output_dir: Path) -> Optional[Dict]:
        """下载单个JavaScript文件"""
        try:
            # 创建新页面用于下载
            page = await self.context.new_page()
            
            # 直接访问JS文件URL
            response = await page.goto(url, timeout=10000)
            
            if response and response.status == 200:
                content = await response.text()
                
                # 生成文件路径
                parsed_url = urlparse(url)
                filename = Path(parsed_url.path).name or 'script.js'
                if not filename.endswith('.js'):
                    filename += '.js'
                file_path = output_dir / filename
                
                # 保存文件
                with open(file_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                
                self.logger.info(f"下载成功: {url} -> {file_path}")
                
                await page.close()
                return {
                    'url': url,
                    'path': str(file_path),
                    'size': len(content.encode('utf-8'))
                }
            else:
                self.logger.warning(f"下载失败: {url} (状态码: {response.status if response else 'None'})")
                
            await page.close()
            
        except Exception as e:
            self.logger.error(f"下载JS文件失败 {url}: {e}")
        
        return None


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