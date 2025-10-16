"""
静态JavaScript文件爬取模块
"""
import re
import time
import requests
import urllib3
from pathlib import Path
from typing import Set, List, Dict, Any
from urllib.parse import urljoin, urlparse
from bs4 import BeautifulSoup
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from tqdm import tqdm
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
import queue

from ..core.config import (
    REQUEST_TIMEOUT, MAX_RETRIES, DELAY_BETWEEN_REQUESTS,
    USER_AGENT, MAX_FILE_SIZE, ORIGINAL_DIR, VERIFY_SSL, SSL_WARNINGS
)
from ..utils.utils import (
    is_valid_url, normalize_url, is_supported_file, 
    generate_file_path, convert_to_utf8, get_content_hash
)
from ..utils.logger import get_logger
from ..utils.report_generator import CrawlReportGenerator

logger = get_logger("static_crawler")

class StaticJSCrawler:
    """静态JavaScript和Source Map文件爬取器"""
    
    def __init__(self, target_url: str = None, output_dir: Path = None):
        self.target_url = target_url
        self.output_dir = output_dir or ORIGINAL_DIR
        
        # 禁用SSL警告（如果配置要求）
        if not SSL_WARNINGS:
            urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        
        self.session = self._create_session()
        self.discovered_urls: Set[str] = set()
        self.downloaded_files: List[Dict[str, Any]] = []
        self.failed_downloads: List[Dict[str, Any]] = []
        self.download_queue = queue.Queue()
        self.download_lock = threading.Lock()
        self.download_executor = None
        
        # 去重机制
        self.content_hashes: Set[str] = set()  # 存储已下载文件的内容哈希
        self.hash_to_filename: Dict[str, str] = {}  # 哈希值到文件名的映射
        self.duplicate_count = 0  # 重复文件计数
        self.processed_urls = set()  # 已处理的URL集合
        self.existing_filenames = set()  # 已存在的文件名集合
        
        # 初始化报告生成器
        self.report_generator = CrawlReportGenerator(self.output_dir)
        
    def _create_session(self) -> requests.Session:
        """创建配置好的requests会话"""
        session = requests.Session()
        
        # 设置SSL验证
        session.verify = VERIFY_SSL
        
        # 设置重试策略
        retry_strategy = Retry(
            total=MAX_RETRIES,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        
        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("http://", adapter)
        session.mount("https://", adapter)
        
        # 设置请求头
        session.headers.update({
            'User-Agent': USER_AGENT,
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
            'Accept-Language': 'en-US,en;q=0.5',
            'Accept-Encoding': 'gzip, deflate',
            'Connection': 'keep-alive',
        })
        
        return session
    
    def discover_files(self, url: str) -> Set[str]:
        """发现页面中的JavaScript和Source Map文件URL"""
        file_urls = set()
        
        try:
            logger.info(f"正在分析页面: {url}")
            response = self.session.get(url, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            
            # 解析HTML
            soup = BeautifulSoup(response.content, 'html.parser')
            
            # 查找script标签中的src属性
            script_tags = soup.find_all('script', src=True)
            for script in script_tags:
                src = script.get('src')
                if src:
                    full_url = normalize_url(url, src)
                    if is_supported_file(full_url):
                        file_urls.add(full_url)
                        self.add_to_download_queue(full_url)  # 立即添加到下载队列
                        logger.debug(f"发现JS文件: {full_url}")
            
            # 查找link标签中的JavaScript模块
            link_tags = soup.find_all('link', rel='modulepreload')
            for link in link_tags:
                href = link.get('href')
                if href and is_supported_file(href):
                    full_url = normalize_url(url, href)
                    file_urls.add(full_url)
                    self.add_to_download_queue(full_url)  # 立即添加到下载队列
                    logger.debug(f"发现JS模块: {full_url}")
            
            # 在HTML内容中搜索JavaScript和Source Map文件引用
            file_patterns = [
                r'["\']([^"\']*\.js(?:\?[^"\']*)?)["\']',
                r'["\']([^"\']*\.mjs(?:\?[^"\']*)?)["\']',
                r'["\']([^"\']*\.js\.map(?:\?[^"\']*)?)["\']',
                r'["\']([^"\']*\.map(?:\?[^"\']*)?)["\']',
                r'import\s+.*?from\s+["\']([^"\']+)["\']',
                r'require\s*\(\s*["\']([^"\']+)["\']',
                r'sourceMappingURL=([^\s]+)',
            ]
            
            content = response.text
            for pattern in file_patterns:
                matches = re.findall(pattern, content, re.IGNORECASE)
                for match in matches:
                    if is_supported_file(match):
                        full_url = normalize_url(url, match)
                        if is_valid_url(full_url):
                            file_urls.add(full_url)
                            self.add_to_download_queue(full_url)  # 立即添加到下载队列
                            logger.debug(f"通过正则发现文件: {full_url}")
            
            logger.info(f"在页面 {url} 中发现 {len(file_urls)} 个JS/MAP文件")
            
        except requests.exceptions.HTTPError as e:
            status_code = e.response.status_code if e.response else 0
            if status_code in [403, 404, 429]:
                logger.warning(f"可能遇到反爬虫机制 - HTTP {status_code}: {url}")
                logger.info("建议使用动态爬取模式")
            else:
                logger.error(f"HTTP错误 {status_code}: {url} - {e}")
            raise  # 重新抛出异常，让上层处理
        except requests.exceptions.RequestException as e:
            logger.error(f"网络请求失败 {url}: {e}")
            raise  # 重新抛出异常，让上层处理
        except Exception as e:
            logger.error(f"分析页面失败 {url}: {e}")
            raise  # 重新抛出异常，让上层处理
        
        return file_urls
    
    def download_file(self, url: str) -> bool:
        """下载单个JavaScript或Source Map文件"""
        start_time = time.time()
        try:
            # 检查URL是否已处理
            if url in self.processed_urls:
                logger.info(f"跳过已处理的URL: {url}")
                return False
            
            # 检查文件是否已经存在（基于文件名）
            from ..utils.utils import is_file_already_downloaded
            if is_file_already_downloaded(url, self.start_url, "static"):
                logger.info(f"跳过已存在的文件: {url}")
                self.processed_urls.add(url)
                return False
            
            logger.info(f"正在下载: {url}")
            self.report_generator.add_log(f"开始下载: {url}")
            
            # 标记URL为已处理
            self.processed_urls.add(url)
            
            # 检查文件大小
            head_response = self.session.head(url, timeout=REQUEST_TIMEOUT)
            content_length = head_response.headers.get('content-length')
            if content_length and int(content_length) > MAX_FILE_SIZE:
                error_msg = f'文件过大 ({content_length} bytes)'
                logger.warning(f"文件过大，跳过下载: {url} ({content_length} bytes)")
                
                # 记录失败信息
                failed_info = {
                    'url': url,
                    'error': error_msg,
                    'type': 'static',
                    'download_time': time.time() - start_time
                }
                self.failed_downloads.append(failed_info)
                self.report_generator.add_failed_file(failed_info)
                return False
            
            # 下载文件
            response = self.session.get(url, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            
            # 检查Content-Type（对于.map文件更宽松的检查）
            content_type = response.headers.get('content-type', '').lower()
            if url.endswith('.map') or url.endswith('.js.map'):
                # Source Map文件通常是application/json或text/plain
                if 'json' not in content_type and 'text' not in content_type and 'javascript' not in content_type:
                    logger.warning(f"可能不是Source Map文件: {url} (Content-Type: {content_type})")
                    self.report_generator.add_log(f"警告: 可能不是Source Map文件: {url} (Content-Type: {content_type})", "WARNING")
            else:
                # JavaScript文件的检查
                if 'javascript' not in content_type and 'text' not in content_type:
                    logger.warning(f"可能不是JavaScript文件: {url} (Content-Type: {content_type})")
                    self.report_generator.add_log(f"警告: 可能不是JavaScript文件: {url} (Content-Type: {content_type})", "WARNING")
            
            # 检查内容是否重复
            content_hash = get_content_hash(response.content)
            if content_hash in self.content_hashes:
                existing_filename = self.hash_to_filename.get(content_hash, "未知文件")
                logger.info(f"跳过重复文件: {url} (与 {existing_filename} 内容相同)")
                self.report_generator.add_log(f"跳过重复文件: {url} (与 {existing_filename} 内容相同)")
                self.duplicate_count += 1
                return False
            
            # 生成本地文件路径（避免文件名冲突）
            from ..utils.utils import generate_unique_file_path
            file_path = generate_unique_file_path(url, self.target_url, 'static', self.existing_filenames)
            
            # 记录文件名
            self.existing_filenames.add(file_path.name)
            
            # 保存文件
            with open(file_path, 'wb') as f:
                f.write(response.content)
            
            # 转换编码为UTF-8
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
                'content_type': content_type,
                'type': 'static',
                'download_time': download_time
            }
            self.downloaded_files.append(file_info)
            self.report_generator.add_success_file(file_info)
            
            logger.info(f"下载成功: {url} -> {file_path}")
            return True
            
        except Exception as e:
            download_time = time.time() - start_time
            error_msg = str(e)
            
            # 检查是否是404错误，如果是则记录为警告而不是错误
            if "404" in error_msg or "Not Found" in error_msg:
                logger.warning(f"页面不存在 {url}: {error_msg}")
            else:
                logger.error(f"下载失败 {url}: {error_msg}")
            
            # 记录失败信息
            failed_info = {
                'url': url,
                'error': error_msg,
                'type': 'static',
                'download_time': download_time
            }
            self.failed_downloads.append(failed_info)
            self.report_generator.add_failed_file(failed_info)
            return False
    
    def _download_worker(self, url: str) -> Dict[str, Any]:
        """下载工作线程"""
        success = self.download_file(url)
        return {'url': url, 'success': success}
    
    def start_concurrent_downloads(self, max_workers: int = 4):
        """启动并发下载器"""
        if self.download_executor is None:
            self.download_executor = ThreadPoolExecutor(max_workers=max_workers)
            logger.info(f"启动并发下载器，最大工作线程: {max_workers}")

    def add_to_download_queue(self, url: str):
        """添加URL到下载队列并立即开始下载"""
        if url not in self.discovered_urls:
            self.discovered_urls.add(url)
            if self.download_executor:
                # 立即提交下载任务
                future = self.download_executor.submit(self._download_worker, url)
                logger.debug(f"已提交下载任务: {url}")

    def wait_for_downloads(self):
        """等待所有下载完成"""
        if self.download_executor:
            self.download_executor.shutdown(wait=True)
            self.download_executor = None
            logger.info("所有下载任务已完成")

    def _process_page_worker(self, page_info: tuple) -> Dict[str, Any]:
        """页面处理工作线程"""
        current_url, depth = page_info
        result = {
            'url': current_url,
            'depth': depth,
            'js_urls': set(),
            'new_pages': [],
            'success': False,
            'error': None
        }
        
        try:
            # 发现当前页面的JavaScript和Source Map文件
            file_urls = self.discover_files(current_url)
            result['js_urls'] = file_urls
            result['success'] = True
            
            # 如果深度允许，发现更多页面
            if depth < self.max_depth:
                response = self.session.get(current_url, timeout=REQUEST_TIMEOUT)
                soup = BeautifulSoup(response.content, 'html.parser')
                
                # 查找页面链接
                links = soup.find_all('a', href=True)
                for link in links:
                    href = link.get('href')
                    if href:
                        full_url = normalize_url(current_url, href)
                        parsed = urlparse(full_url)
                        start_parsed = urlparse(self.start_url)
                        
                        # 只访问同域名的页面
                        if parsed.netloc == start_parsed.netloc:
                            result['new_pages'].append((full_url, depth + 1))
            
        except requests.exceptions.HTTPError as e:
            status_code = e.response.status_code if e.response else 0
            error_msg = f"HTTP错误 {status_code}: {current_url}"
            if status_code in [403, 404, 429]:
                error_msg = f"遇到反爬虫机制，无法访问页面: {current_url}"
            result['error'] = error_msg
            logger.error(error_msg)
        except Exception as e:
            result['error'] = f"分析页面失败 {current_url}: {e}"
            logger.error(result['error'])
        
        return result

    def crawl_website(self, start_url: str, max_depth: int = 2, max_workers: int = 4) -> Dict[str, Any]:
        """爬取网站的JavaScript文件"""
        logger.info(f"开始爬取网站: {start_url}")
        logger.info(f"使用 {max_workers} 个并发线程进行页面访问和下载")
        
        # 保存参数供工作线程使用
        self.start_url = start_url
        self.max_depth = max_depth
        
        # 启动并发下载器
        self.start_concurrent_downloads(max_workers=max_workers)
        
        visited_pages = set()
        pages_to_visit = [(start_url, 0)]  # (url, depth)
        
        # 使用线程池并发处理页面
        with ThreadPoolExecutor(max_workers=max_workers) as page_executor:
            while pages_to_visit:
                # 准备当前批次的页面
                current_batch = []
                batch_size = min(max_workers, len(pages_to_visit))
                
                for _ in range(batch_size):
                    if pages_to_visit:
                        page_info = pages_to_visit.pop(0)
                        current_url, depth = page_info
                        
                        if current_url not in visited_pages and depth <= max_depth:
                            visited_pages.add(current_url)
                            current_batch.append(page_info)
                
                if not current_batch:
                    break
                
                # 并发处理当前批次的页面
                logger.info(f"并发处理 {len(current_batch)} 个页面...")
                future_to_page = {
                    page_executor.submit(self._process_page_worker, page_info): page_info 
                    for page_info in current_batch
                }
                
                # 收集结果
                for future in as_completed(future_to_page):
                    page_info = future_to_page[future]
                    try:
                        result = future.result()
                        
                        if result['success']:
                            # 添加发现的JS文件到下载队列
                            for js_url in result['js_urls']:
                                if js_url not in self.discovered_urls:
                                    self.discovered_urls.add(js_url)
                                    if self.download_executor:
                                        self.download_executor.submit(self._download_worker, js_url)
                            
                            # 添加新发现的页面到待访问列表
                            for new_page in result['new_pages']:
                                if new_page[0] not in visited_pages:
                                    pages_to_visit.append(new_page)
                        else:
                            # 如果是主页面失败，直接抛出异常
                            if page_info[0] == start_url:
                                raise Exception(f"无法访问目标网站: {result['error']}")
                    
                    except Exception as e:
                        logger.error(f"处理页面时出错: {e}")
                        # 如果是主页面失败，直接抛出异常
                        if page_info[0] == start_url:
                            raise Exception(f"无法访问目标网站: {e}")
                
                # 添加延迟避免过于频繁的请求
                if pages_to_visit:
                    time.sleep(DELAY_BETWEEN_REQUESTS)
        
        # 等待所有并发下载完成
        logger.info(f"页面访问完成，发现 {len(self.discovered_urls)} 个JavaScript文件")
        logger.info("等待并发下载完成...")
        self.wait_for_downloads()
        
        # 生成并保存详细报告
        logger.info("正在生成爬取报告...")
        self.report_generator.add_log(f"爬取完成 - 发现 {len(self.discovered_urls)} 个文件，成功下载 {len(self.downloaded_files)} 个，跳过重复文件 {self.duplicate_count} 个")
        self.report_generator.save_all_reports()
        self.report_generator.print_summary()
        
        # 返回统计信息
        return {
            'total_discovered': len(self.discovered_urls),
            'successful_downloads': len(self.downloaded_files),
            'failed_downloads': len(self.failed_downloads),
            'duplicate_files': self.duplicate_count,
            'visited_pages': len(visited_pages),
            'downloaded_files': self.downloaded_files,
            'failed_files': self.failed_downloads
        }
    
    def get_statistics(self) -> Dict[str, Any]:
        """获取爬取统计信息"""
        total_size = sum(file_info['size'] for file_info in self.downloaded_files)
        
        return {
            'discovered_urls': len(self.discovered_urls),
            'downloaded_files': len(self.downloaded_files),
            'failed_downloads': len(self.failed_downloads),
            'duplicate_files': self.duplicate_count,
            'total_size': total_size,
            'success_rate': len(self.downloaded_files) / len(self.discovered_urls) * 100 if self.discovered_urls else 0
        }