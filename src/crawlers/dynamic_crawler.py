"""
动态JavaScript文件捕获模块
"""
import json
import time
import requests
from pathlib import Path
from typing import Set, List, Dict, Any, Optional
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, WebDriverException
from webdriver_manager.chrome import ChromeDriverManager
from tqdm import tqdm

from ..core.config import (
    SELENIUM_TIMEOUT, PAGE_LOAD_TIMEOUT, IMPLICIT_WAIT, HEADLESS_MODE,
    USER_AGENT, ORIGINAL_DIR, MAX_FILE_SIZE, VERIFY_SSL
)
from ..utils.utils import (
    is_valid_url, normalize_url, is_supported_file,
    generate_file_path, format_file_size, convert_to_utf8, get_content_hash
)
from ..utils.logger import get_logger
from ..utils.report_generator import CrawlReportGenerator

logger = get_logger("dynamic_crawler")

class DynamicJSCrawler:
    """动态JavaScript和Source Map文件捕获器"""
    
    def __init__(self, target_url: str = None, output_dir: Path = None, existing_file_hashes: Dict[str, str] = None):
        self.target_url = target_url
        self.output_dir = output_dir or ORIGINAL_DIR
        self.driver: Optional[webdriver.Chrome] = None
        self.captured_requests: List[Dict[str, Any]] = []
        self.downloaded_files: List[Dict[str, Any]] = []
        self.failed_downloads: List[Dict[str, Any]] = []
        self.js_urls: Set[str] = set()
        
        # 去重机制
        self.content_hashes: Set[str] = set()  # 存储已下载文件的内容哈希
        self.hash_to_filename: Dict[str, str] = {}  # 哈希值到文件名的映射
        self.duplicate_count = 0  # 重复文件计数
        self.cross_mode_duplicate_count = 0  # 跨模式重复文件计数
        self.processed_urls = set()  # 已处理的URL集合
        self.existing_filenames = set()  # 已存在的文件名集合
        
        # 初始化已有文件哈希（用于跨模式去重）
        self.static_file_hashes = set()  # 存储来自静态爬取的文件哈希
        if existing_file_hashes:
            self.content_hashes.update(existing_file_hashes.keys())
            self.hash_to_filename.update(existing_file_hashes)
            self.static_file_hashes.update(existing_file_hashes.keys())
            logger.info(f"动态爬虫加载了 {len(existing_file_hashes)} 个已有文件哈希，用于跨模式去重")
        
        # 初始化报告生成器
        self.report_generator = CrawlReportGenerator(self.output_dir)
        
    def _setup_driver(self) -> webdriver.Chrome:
        """设置Chrome WebDriver"""
        chrome_options = Options()
        
        if HEADLESS_MODE:
            chrome_options.add_argument("--headless")
        
        # 基础配置
        chrome_options.add_argument(f"--user-agent={USER_AGENT}")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--disable-web-security")
        chrome_options.add_argument("--allow-running-insecure-content")
        chrome_options.add_argument("--disable-extensions")
        chrome_options.add_argument("--disable-plugins")
        chrome_options.add_argument("--disable-images")
        
        # 启用网络日志
        chrome_options.add_argument("--enable-logging")
        chrome_options.add_argument("--log-level=0")
        chrome_options.add_experimental_option("useAutomationExtension", False)
        chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
        
        # 启用性能日志以捕获网络请求
        chrome_options.set_capability('goog:loggingPrefs', {
            'performance': 'ALL',
            'browser': 'ALL'
        })
        
        driver = None
        
        # 方法1: 尝试使用系统已安装的ChromeDriver
        try:
            logger.info("尝试使用系统ChromeDriver...")
            driver = webdriver.Chrome(options=chrome_options)
            driver.set_page_load_timeout(PAGE_LOAD_TIMEOUT)
            driver.implicitly_wait(IMPLICIT_WAIT)
            
            # 启用网络域
            driver.execute_cdp_cmd('Network.enable', {})
            driver.execute_cdp_cmd('Runtime.enable', {})
            
            logger.info("系统ChromeDriver初始化成功")
            return driver
            
        except Exception as e:
            logger.warning(f"系统ChromeDriver失败: {e}")
            if driver:
                try:
                    driver.quit()
                except:
                    pass
                driver = None
        
        # 方法2: 尝试使用webdriver-manager
        try:
            logger.info("尝试使用webdriver-manager下载ChromeDriver...")
            import platform
            import os
            
            # 检测系统架构
            arch = platform.machine().lower()
            system = platform.system().lower()
            logger.info(f"检测到系统架构: {system} {arch}")
            
            # 使用webdriver-manager下载ChromeDriver
            driver_manager = ChromeDriverManager()
            driver_path = driver_manager.install()
            logger.info(f"ChromeDriver路径: {driver_path}")
            
            # 检查驱动文件是否可执行
            if not os.access(driver_path, os.X_OK):
                logger.warning(f"ChromeDriver文件不可执行，尝试修复权限: {driver_path}")
                os.chmod(driver_path, 0o755)
            
            service = Service(driver_path)
            driver = webdriver.Chrome(service=service, options=chrome_options)
            driver.set_page_load_timeout(PAGE_LOAD_TIMEOUT)
            driver.implicitly_wait(IMPLICIT_WAIT)
            
            # 启用网络域
            driver.execute_cdp_cmd('Network.enable', {})
            driver.execute_cdp_cmd('Runtime.enable', {})
            
            logger.info("webdriver-manager ChromeDriver初始化成功")
            return driver
            
        except Exception as e:
            logger.warning(f"webdriver-manager失败: {e}")
            if driver:
                try:
                    driver.quit()
                except:
                    pass
                driver = None
        
        # 方法3: 尝试使用Selenium Manager (Selenium 4.6+)
        try:
            logger.info("尝试使用Selenium Manager...")
            # 不指定service，让Selenium自己管理
            driver = webdriver.Chrome(options=chrome_options)
            driver.set_page_load_timeout(PAGE_LOAD_TIMEOUT)
            driver.implicitly_wait(IMPLICIT_WAIT)
            
            # 启用网络域
            driver.execute_cdp_cmd('Network.enable', {})
            driver.execute_cdp_cmd('Runtime.enable', {})
            
            logger.info("Selenium Manager ChromeDriver初始化成功")
            return driver
            
        except Exception as e:
            logger.warning(f"Selenium Manager失败: {e}")
            if driver:
                try:
                    driver.quit()
                except:
                    pass
                driver = None
        
        # 所有方法都失败了
        error_msg = "所有ChromeDriver初始化方法都失败了"
        logger.error(error_msg)
        logger.error("请尝试以下解决方案：")
        logger.error("1. 确保Chrome浏览器已安装")
        logger.error("2. 检查网络连接")
        logger.error("3. 手动下载ChromeDriver并添加到PATH")
        logger.error("4. 更新Chrome浏览器到最新版本")
        
        raise Exception(error_msg)
    
    def _extract_js_from_logs(self) -> Set[str]:
        """从浏览器日志中提取JavaScript文件URL"""
        js_urls = set()
        
        try:
            # 获取性能日志
            logs = self.driver.get_log('performance')
            
            for log in logs:
                message = json.loads(log['message'])
                
                if message['message']['method'] == 'Network.responseReceived':
                    response = message['message']['params']['response']
                    url = response['url']
                    mime_type = response.get('mimeType', '')
                    
                    # 检查是否为支持的文件类型
                    if (is_supported_file(url) or 
                        'javascript' in mime_type.lower() or
                        'application/javascript' in mime_type.lower() or
                        'text/javascript' in mime_type.lower()):
                        js_urls.add(url)
                        logger.debug(f"从网络日志发现文件: {url}")
                
                elif message['message']['method'] == 'Network.requestWillBeSent':
                    request = message['message']['params']['request']
                    url = request['url']
                    
                    if is_supported_file(url):
                        js_urls.add(url)
                        logger.debug(f"从请求日志发现文件: {url}")
        
        except Exception as e:
            logger.error(f"从日志提取JS文件失败: {e}")
        
        return js_urls
    
    def _capture_xhr_requests(self) -> Set[str]:
        """捕获XHR/Fetch请求中的JavaScript文件"""
        js_urls = set()
        
        try:
            # 注入JavaScript代码来监控XHR和Fetch请求
            monitor_script = """
            window.capturedRequests = window.capturedRequests || [];
            
            // 监控XMLHttpRequest
            (function() {
                var originalOpen = XMLHttpRequest.prototype.open;
                XMLHttpRequest.prototype.open = function(method, url) {
                    this._url = url;
                    return originalOpen.apply(this, arguments);
                };
                
                var originalSend = XMLHttpRequest.prototype.send;
                XMLHttpRequest.prototype.send = function() {
                    var xhr = this;
                    xhr.addEventListener('load', function() {
                        if (xhr._url && (xhr._url.includes('.js') || 
                            xhr.getResponseHeader('content-type').includes('javascript'))) {
                            window.capturedRequests.push({
                                url: xhr._url,
                                type: 'xhr',
                                status: xhr.status
                            });
                        }
                    });
                    return originalSend.apply(this, arguments);
                };
            })();
            
            // 监控Fetch API
            (function() {
                var originalFetch = window.fetch;
                window.fetch = function() {
                    var url = arguments[0];
                    return originalFetch.apply(this, arguments).then(function(response) {
                        if (url && (url.includes('.js') || 
                            response.headers.get('content-type').includes('javascript'))) {
                            window.capturedRequests.push({
                                url: url,
                                type: 'fetch',
                                status: response.status
                            });
                        }
                        return response;
                    });
                };
            })();
            """
            
            self.driver.execute_script(monitor_script)
            
            # 等待一段时间让页面加载完成
            time.sleep(3)
            
            # 获取捕获的请求
            captured = self.driver.execute_script("return window.capturedRequests || [];")
            
            for request in captured:
                url = request.get('url')
                if url and is_supported_file(url):
                    js_urls.add(url)
                    logger.debug(f"从XHR/Fetch捕获文件: {url}")
        
        except Exception as e:
            logger.error(f"捕获XHR请求失败: {e}")
        
        return js_urls
    
    def _trigger_dynamic_content(self, base_url: str):
        """触发动态内容加载"""
        try:
            # 滚动页面以触发懒加载
            self.driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
            time.sleep(2)
            
            # 点击可能触发动态加载的元素
            clickable_selectors = [
                "button", "a[href='#']", ".load-more", ".show-more", 
                "[onclick]", "[data-toggle]", ".tab", ".menu-item"
            ]
            
            for selector in clickable_selectors:
                try:
                    elements = self.driver.find_elements(By.CSS_SELECTOR, selector)
                    for element in elements[:3]:  # 限制点击数量
                        try:
                            if element.is_displayed() and element.is_enabled():
                                self.driver.execute_script("arguments[0].click();", element)
                                time.sleep(1)
                        except Exception:
                            continue
                except Exception:
                    continue
            
            # 触发常见事件
            events = ['mouseover', 'focus', 'change']
            for event in events:
                try:
                    self.driver.execute_script(f"""
                        var elements = document.querySelectorAll('*');
                        for (var i = 0; i < Math.min(elements.length, 10); i++) {{
                            var event = new Event('{event}');
                            elements[i].dispatchEvent(event);
                        }}
                    """)
                    time.sleep(0.5)
                except Exception:
                    continue
                    
        except Exception as e:
            logger.error(f"触发动态内容失败: {e}")
    
    def _download_file(self, url: str) -> bool:
        """下载JavaScript和Source Map文件"""
        start_time = time.time()
        try:
            # 检查URL是否已处理
            if url in self.processed_urls:
                logger.info(f"跳过已处理的URL: {url}")
                return False
            
            # 检查文件是否已经存在（基于文件名）
            from ..utils.utils import is_file_already_downloaded
            if is_file_already_downloaded(url, self.target_url, "dynamic"):
                logger.info(f"跳过已存在的动态文件: {url}")
                self.processed_urls.add(url)
                return False
            
            logger.info(f"正在下载动态文件: {url}")
            self.report_generator.add_log(f"开始下载动态文件: {url}")
            
            # 标记URL为已处理
            self.processed_urls.add(url)
            
            # 使用requests下载文件
            session = requests.Session()
            session.headers.update({'User-Agent': USER_AGENT})
            session.verify = VERIFY_SSL  # 设置SSL验证
            
            # 检查文件大小
            try:
                head_response = session.head(url, timeout=30)
                content_length = head_response.headers.get('content-length')
                if content_length and int(content_length) > MAX_FILE_SIZE:
                    error_msg = f'文件过大 ({content_length} bytes)'
                    logger.warning(f"动态文件过大，跳过: {url}")
                    
                    # 记录失败信息
                    failed_info = {
                        'url': url,
                        'error': error_msg,
                        'type': 'dynamic',
                        'download_time': time.time() - start_time
                    }
                    self.failed_downloads.append(failed_info)
                    self.report_generator.add_failed_file(failed_info)
                    return False
            except Exception:
                pass  # 继续尝试下载
            
            # 下载文件
            response = session.get(url, timeout=30)
            response.raise_for_status()
            
            # 检查内容是否重复
            content_hash = get_content_hash(response.content)
            if content_hash in self.content_hashes:
                existing_filename = self.hash_to_filename.get(content_hash, "未知文件")
                
                # 判断是跨模式去重还是模式内去重
                is_cross_mode = content_hash in self.static_file_hashes
                
                if is_cross_mode:
                    logger.info(f"跳过跨模式重复文件: {url} (与静态爬取的 {existing_filename} 内容相同)")
                    self.report_generator.add_log(f"跳过跨模式重复文件: {url} (与静态爬取的 {existing_filename} 内容相同)")
                    self.cross_mode_duplicate_count += 1
                else:
                    logger.info(f"跳过重复文件: {url} (与 {existing_filename} 内容相同)")
                    self.report_generator.add_log(f"跳过重复文件: {url} (与 {existing_filename} 内容相同)")
                    self.duplicate_count += 1
                return False
            
            # 生成本地文件路径（避免文件名冲突）
            from ..utils.utils import generate_unique_file_path
            file_path = generate_unique_file_path(url, self.target_url, 'dynamic', self.existing_filenames)
            
            # 记录文件名
            self.existing_filenames.add(file_path.name)
            
            # 确保目录存在
            file_path.parent.mkdir(parents=True, exist_ok=True)
            
            # 保存文件
            with open(file_path, 'wb') as f:
                f.write(response.content)
            
            # 转换编码
            convert_to_utf8(file_path)
            
            # 更新去重信息
            self.content_hashes.add(content_hash)
            self.hash_to_filename[content_hash] = file_path.name
            
            download_time = time.time() - start_time
            
            # 记录成功下载
            file_info = {
                'url': url,
                'file_path': str(file_path),
                'local_path': str(file_path),  # 保持向后兼容
                'size': len(response.content),
                'content_type': response.headers.get('content-type', ''),
                'type': 'dynamic',
                'download_time': download_time
            }
            self.downloaded_files.append(file_info)
            self.report_generator.add_success_file(file_info)
            
            logger.info(f"动态文件下载成功: {url} -> {file_path}")
            return True
            
        except Exception as e:
            download_time = time.time() - start_time
            error_msg = str(e)
            logger.error(f"下载动态文件失败 {url}: {error_msg}")
            
            # 记录失败信息
            failed_info = {
                'url': url,
                'error': error_msg,
                'type': 'dynamic',
                'download_time': download_time
            }
            self.failed_downloads.append(failed_info)
            self.report_generator.add_failed_file(failed_info)
            return False
    
    def crawl_dynamic_js(self, url: str, wait_time: int = 10) -> Dict[str, Any]:
        """爬取动态加载的JavaScript文件"""
        logger.info(f"开始动态爬取: {url}")
        
        try:
            # 初始化WebDriver
            self.driver = self._setup_driver()
            
            # 访问页面
            logger.info(f"正在访问页面: {url}")
            self.driver.get(url)
            
            # 等待页面初始加载
            time.sleep(3)
            
            # 从初始日志中提取JS文件
            initial_js = self._extract_js_from_logs()
            self.js_urls.update(initial_js)
            
            # 触发动态内容加载
            logger.info("正在触发动态内容加载...")
            self._trigger_dynamic_content(url)
            
            # 等待动态内容加载
            time.sleep(wait_time)
            
            # 再次提取JS文件
            dynamic_js = self._extract_js_from_logs()
            self.js_urls.update(dynamic_js)
            
            # 捕获XHR/Fetch请求
            xhr_js = self._capture_xhr_requests()
            self.js_urls.update(xhr_js)
            
            logger.info(f"发现 {len(self.js_urls)} 个动态文件")
            
            # 下载所有发现的文件
            with tqdm(total=len(self.js_urls), desc="下载动态文件", unit="文件") as pbar:
                for js_url in self.js_urls:
                    success = self._download_file(js_url)
                    pbar.set_postfix({
                        '成功': len(self.downloaded_files),
                        '失败': len(self.failed_downloads)
                    })
                    pbar.update(1)
                    time.sleep(1)  # 避免请求过快
            
            # 生成并保存报告
            self.report_generator.add_log(f"动态爬取完成，发现 {len(self.js_urls)} 个文件，成功下载 {len(self.downloaded_files)} 个，跳过重复文件 {self.duplicate_count} 个")
            self.report_generator.save_all_reports()
            
            # 打印报告摘要
            summary = self.report_generator.generate_summary_report()
            logger.info(f"动态爬取报告摘要: {summary['crawl_summary']}")
            
            return {
                'total_discovered': len(self.js_urls),
                'successful_downloads': len(self.downloaded_files),
                'failed_downloads': len(self.failed_downloads),
                'duplicate_files': self.duplicate_count,
                'cross_mode_duplicated_files': self.cross_mode_duplicate_count,
                'downloaded_files': self.downloaded_files,
                'failed_files': self.failed_downloads
            }
            
        except Exception as e:
            logger.error(f"动态爬取失败: {e}")
            # 记录异常到报告
            self.report_generator.add_log(f"动态爬取异常: {str(e)}")
            self.report_generator.save_all_reports()
            
            return {
                'total_discovered': 0,
                'successful_downloads': 0,
                'failed_downloads': 0,
                'downloaded_files': [],
                'failed_files': [{'url': url, 'reason': str(e), 'type': 'dynamic'}]
            }
        
        finally:
            if self.driver:
                try:
                    self.driver.quit()
                except Exception as e:
                    logger.error(f"关闭WebDriver失败: {e}")
    
    def get_statistics(self) -> Dict[str, Any]:
        """获取动态爬取统计信息"""
        total_size = sum(file_info['size'] for file_info in self.downloaded_files)
        
        return {
            'discovered_urls': len(self.js_urls),
            'downloaded_files': len(self.downloaded_files),
            'failed_downloads': len(self.failed_downloads),
            'duplicate_files': self.duplicate_count,
            'total_size': total_size,
            'success_rate': len(self.downloaded_files) / len(self.js_urls) * 100 if self.js_urls else 0
        }