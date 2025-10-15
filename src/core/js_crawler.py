"""
JavaScript文件爬取和反混淆主程序
"""
import sys
import time
import argparse
from pathlib import Path
from typing import Dict, Any, Optional
from datetime import datetime

from .config import ensure_directories, get_directory_structure, SAVE_CHECKPOINT_INTERVAL, BROWSER_ENGINE, USE_EMBEDDED_BROWSER
from ..utils.logger import setup_logger, get_logger
from ..crawlers.static_crawler import StaticJSCrawler
from ..crawlers.dynamic_crawler import DynamicJSCrawler
from .deobfuscator import JSDeobfuscator
from ..utils.utils import save_checkpoint, load_checkpoint, format_file_size, calculate_file_hash
from ..utils.parallel_similarity_analyzer import ParallelDeduplicationProcessor

# 根据配置选择浏览器引擎
if USE_EMBEDDED_BROWSER and BROWSER_ENGINE == "playwright":
    try:
        from ..crawlers.playwright_crawler import PlaywrightCrawler
        PLAYWRIGHT_AVAILABLE = True
    except ImportError:
        PLAYWRIGHT_AVAILABLE = False
        # 延迟logger初始化，避免循环导入
        def _warn_playwright_unavailable():
            setup_logger()
            logger = get_logger("main")
            _get_logger().warning("Playwright未安装，将使用传统Selenium方式")
        _warn_playwright_unavailable()
else:
    PLAYWRIGHT_AVAILABLE = False

# 延迟logger初始化函数
def _get_logger():
    setup_logger()
    return get_logger("main")

class JSCrawler:
    """JavaScript爬取器 - 面向用户的主要接口"""
    
    def __init__(self, target_url: str):
        self.target_url = target_url
        self.dirs = get_directory_structure(target_url)
        self.output_dir = self.dirs['target_output_dir']  # 添加output_dir属性
        self.manager = JSCrawlerManager(target_url)
    
    def crawl(self, depth: int = 2, wait_time: int = 3, max_workers: int = 2, playwright_tabs: int = 4, 
              headless: bool = True, mode: str = 'all', resume: bool = False,
              similarity_enabled: bool = True, similarity_threshold: float = 0.8,
              similarity_workers: int = None, auto_similarity: bool = True) -> Dict[str, Any]:
        """执行爬取操作"""
        return self.manager.run(
            url=self.target_url,
            max_depth=depth,
            wait_time=wait_time,
            max_workers=max_workers,
            playwright_tabs=playwright_tabs,
            headless=headless,
            mode=mode,
            resume=resume,
            similarity_enabled=similarity_enabled,
            similarity_threshold=similarity_threshold,
            similarity_workers=similarity_workers,
            auto_similarity=auto_similarity
        )

class JSCrawlerManager:
    """JavaScript爬取管理器"""
    
    def __init__(self, target_url: str):
        self.target_url = target_url
        self.dirs = get_directory_structure(target_url)
        ensure_directories(target_url)  # 确保目录存在
        self.checkpoint_file = self.dirs['checkpoints_dir'] / 'crawler_checkpoint.json'
        self.start_time = time.time()
        # 传递output_dir参数给爬虫，以便生成报告
        self.static_crawler = StaticJSCrawler(target_url, self.dirs['target_output_dir'])
        self.dynamic_crawler = DynamicJSCrawler(target_url, self.dirs['target_output_dir'])
        self.deobfuscator = JSDeobfuscator()
        self.checkpoint_data = {}
        
    def load_checkpoint(self) -> Optional[Dict[str, Any]]:
        """加载检查点数据"""
        try:
            checkpoint = load_checkpoint(self.checkpoint_file)
            if checkpoint:
                _get_logger().info("发现检查点文件，将从上次中断处继续")
                return checkpoint
        except Exception as e:
            _get_logger().error(f"加载检查点失败: {e}")
        return None
    
    def save_checkpoint(self, data: Dict[str, Any]) -> bool:
        """保存检查点数据"""
        try:
            self.checkpoint_data.update(data)
            return save_checkpoint(self.checkpoint_data, self.checkpoint_file)
        except Exception as e:
            _get_logger().error(f"保存检查点失败: {e}")
            return False
    
    def _is_anti_crawler_detected(self, results: Dict[str, Any]) -> bool:
        """检测是否遇到反爬虫机制"""
        # 如果访问了页面但没有发现任何JS文件，可能是反爬虫
        if (results.get('visited_pages', 0) > 0 and 
            results.get('total_discovered', 0) == 0 and
            results.get('successful_downloads', 0) == 0):
            return True
        
        # 如果发现的文件数量很少且下载失败率很高
        total_discovered = results.get('total_discovered', 0)
        failed_downloads = results.get('failed_downloads', 0)
        if total_discovered > 0 and failed_downloads / total_discovered > 0.8:
            return True
            
        return False
    
    def _is_anti_crawler_error(self, error: Exception) -> bool:
        """检测是否是反爬虫相关的错误"""
        error_str = str(error).lower()
        anti_crawler_indicators = [
            '404', '403', '401', 'forbidden', 'not found',
            'access denied', 'blocked', 'captcha', 'verification',
            'too many requests', 'rate limit', 'cloudflare'
        ]
        
        return any(indicator in error_str for indicator in anti_crawler_indicators)
    
    def _collect_downloaded_file_hashes(self) -> Dict[str, str]:
        """收集已下载文件的哈希值，用于跨模式去重"""
        file_hashes = {}
        original_dir = self.dirs['original_dir']
        
        if original_dir.exists():
            for js_file in original_dir.rglob('*.js'):
                try:
                    file_hash = calculate_file_hash(js_file)
                    file_hashes[file_hash] = str(js_file.name)
                    _get_logger().debug(f"收集文件哈希: {js_file.name} -> {file_hash}")
                except Exception as e:
                    _get_logger().warning(f"计算文件哈希失败 {js_file}: {e}")
        
        _get_logger().info(f"收集到 {len(file_hashes)} 个已下载文件的哈希值")
        return file_hashes
    
    def crawl_static_js(self, url: str, max_depth: int = 2, max_workers: int = 4, resume: bool = False) -> Dict[str, Any]:
        """爬取静态JavaScript文件"""
        _get_logger().info("=" * 60)
        _get_logger().info("开始静态JavaScript文件爬取")
        _get_logger().info("=" * 60)
        
        try:
            # 检查是否需要恢复
            if resume and 'static_completed' in self.checkpoint_data:
                _get_logger().info("静态爬取已完成，跳过此步骤")
                return self.checkpoint_data.get('static_results', {})
            
            # 执行静态爬取
            results = self.static_crawler.crawl_website(url, max_depth, max_workers)
            
            # 检查是否遇到反爬虫机制
            if self._is_anti_crawler_detected(results):
                _get_logger().warning("检测到反爬虫机制，静态爬取失败")
                _get_logger().info("将自动切换到动态爬取模式...")
                # 标记静态爬取失败，但不保存为完成状态
                self.save_checkpoint({
                    'static_failed_anti_crawler': True,
                    'static_results': results
                })
                return results
            
            # 保存检查点
            self.save_checkpoint({
                'static_completed': True,
                'static_results': results
            })
            
            # 输出统计信息
            _get_logger().info(f"静态爬取完成:")
            _get_logger().info(f"  - 发现文件数: {results['total_discovered']}")
            _get_logger().info(f"  - 成功下载: {results['successful_downloads']}")
            _get_logger().info(f"  - 失败下载: {results['failed_downloads']}")
            _get_logger().info(f"  - 访问页面数: {results['visited_pages']}")
            
            return results
            
        except Exception as e:
            _get_logger().error(f"静态爬取失败: {e}")
            # 检查是否是反爬虫相关的错误
            if self._is_anti_crawler_error(e):
                _get_logger().warning("检测到反爬虫机制导致的错误")
                _get_logger().info("将自动切换到动态爬取模式...")
                self.save_checkpoint({
                    'static_failed_anti_crawler': True
                })
            
            return {
                'total_discovered': 0,
                'successful_downloads': 0,
                'failed_downloads': 0,
                'visited_pages': 0,
                'downloaded_files': [],
                'failed_files': []
            }
    
    def crawl_dynamic_js(self, url: str, wait_time: int = 10, playwright_tabs: int = 4, headless: bool = True, resume: bool = False) -> Dict[str, Any]:
        """爬取动态JavaScript文件"""
        _get_logger().info("=" * 60)
        _get_logger().info("开始动态JavaScript文件爬取")
        _get_logger().info("=" * 60)
        
        try:
            # 检查是否需要恢复
            if resume and 'dynamic_completed' in self.checkpoint_data:
                _get_logger().info("动态爬取已完成，跳过此步骤")
                return self.checkpoint_data.get('dynamic_results', {})
            
            # 收集已下载文件的哈希值，用于跨模式去重
            existing_file_hashes = self._collect_downloaded_file_hashes()
            
            # 根据配置选择爬取方式
            if USE_EMBEDDED_BROWSER and PLAYWRIGHT_AVAILABLE:
                _get_logger().info(f"使用Playwright进行动态爬取，最大标签页数: {playwright_tabs}，无头模式: {headless}")
                results = self._crawl_with_playwright(url, wait_time, playwright_tabs, headless, existing_file_hashes)
            else:
                # 使用传统Selenium方式
                _get_logger().info("使用传统Selenium进行动态爬取")
                # 为传统Selenium动态爬虫设置静态文件哈希值
                if existing_file_hashes:
                    self.dynamic_crawler.content_hashes.update(existing_file_hashes.keys())
                    self.dynamic_crawler.hash_to_filename.update(existing_file_hashes)
                    self.dynamic_crawler.static_file_hashes.update(existing_file_hashes.keys())
                    _get_logger().info(f"为传统动态爬虫设置了 {len(existing_file_hashes)} 个静态文件哈希")
                results = self.dynamic_crawler.crawl_dynamic_js(url, wait_time)
            
            # 保存检查点
            self.save_checkpoint({
                'dynamic_completed': True,
                'dynamic_results': results
            })
            
            # 输出统计信息
            _get_logger().info(f"动态爬取完成:")
            _get_logger().info(f"  - 发现文件数: {results['total_discovered']}")
            _get_logger().info(f"  - 成功下载: {results['successful_downloads']}")
            _get_logger().info(f"  - 失败下载: {results['failed_downloads']}")
            
            duplicated_files = results.get('duplicated_files', 0)
            cross_mode_duplicated = results.get('cross_mode_duplicated_files', 0)
            total_duplicated = duplicated_files + cross_mode_duplicated
            
            if total_duplicated > 0:
                _get_logger().info(f"  - 重复文件: {total_duplicated} (已跳过)")
                if cross_mode_duplicated > 0:
                    _get_logger().info(f"    - 模式内去重: {duplicated_files}")
                    _get_logger().info(f"    - 跨模式去重: {cross_mode_duplicated}")
                    _get_logger().info(f"  - 去重效果: 节省了 {total_duplicated} 个重复文件的下载 (其中 {cross_mode_duplicated} 个为跨模式去重)")
                else:
                    _get_logger().info(f"  - 去重效果: 节省了 {total_duplicated} 个重复文件的下载")
            
            return results
            
        except Exception as e:
            _get_logger().error(f"动态爬取失败: {e}")
            return {
                'total_discovered': 0,
                'successful_downloads': 0,
                'failed_downloads': 0,
                'downloaded_files': [],
                'failed_files': []
            }
    
    def _crawl_with_playwright(self, url: str, wait_time: int = 10, playwright_tabs: int = 4, headless: bool = True, existing_file_hashes: Dict[str, str] = None) -> Dict[str, Any]:
        """使用Playwright进行动态爬取"""
        import asyncio
        from .config import PLAYWRIGHT_BROWSER
        
        async def async_crawl():
            async with PlaywrightCrawler(
                target_url=url,
                max_depth=2, 
                wait_time=wait_time,
                max_workers=playwright_tabs,  # 传递Playwright标签页数量控制参数
                browser_type=PLAYWRIGHT_BROWSER,
                headless=headless,  # 传递无头模式参数
                existing_file_hashes=existing_file_hashes  # 传递已有文件哈希
            ) as crawler:
                stats = await crawler.crawl_website(url, self.dirs['original_dir'])
                return {
                    'total_discovered': stats['js_files_found'],
                    'successful_downloads': stats['js_files_downloaded'],
                    'failed_downloads': stats['js_files_failed'],
                    'duplicated_files': stats.get('duplicated_files', 0),
                    'cross_mode_duplicated_files': stats.get('cross_mode_duplicated_files', 0),
                    'downloaded_files': [],  # Playwright处理文件列表的方式不同
                    'failed_files': [],
                    'total_size': stats['total_size']
                }
        
        # 运行异步爬取
        return asyncio.run(async_crawl())
    
    def run_similarity_analysis(self, similarity_threshold: float = 0.8, 
                               similarity_workers: int = None, resume: bool = False) -> Dict[str, Any]:
        """运行智能相似度分析"""
        _get_logger().info("=" * 60)
        _get_logger().info("开始智能相似度分析")
        _get_logger().info("=" * 60)
        
        try:
            # 检查是否需要恢复
            if resume and 'similarity_completed' in self.checkpoint_data:
                _get_logger().info("相似度分析已完成，跳过此步骤")
                return self.checkpoint_data.get('similarity_results', {})
            
            # 检查decode目录是否存在
            decode_dir = self.dirs['decrypted_dir']
            if not decode_dir.exists() or not any(decode_dir.glob('*.js')):
                _get_logger().warning("未找到反编译文件，跳过相似度分析")
                return {
                    'success': True,
                    'total_files': 0,
                    'unique_files': 0,
                    'similar_groups': 0,
                    'exact_duplicate_groups': 0,
                    'processing_time_seconds': 0,
                    'output_dir': None
                }
            
            # 创建输出目录
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            similarity_output_dir = self.dirs['target_output_dir'] / f'similarity_analysis_{timestamp}'
            
            # 创建并运行相似度分析器
            processor = ParallelDeduplicationProcessor(
                similarity_threshold=similarity_threshold,
                max_workers=similarity_workers
            )
            
            _get_logger().info(f"开始分析 {decode_dir} 中的反编译文件")
            _get_logger().info(f"相似度阈值: {similarity_threshold}")
            _get_logger().info(f"并行进程数: {similarity_workers or 'auto'}")
            
            # 执行分析
            report = processor.process_directory(str(decode_dir), str(similarity_output_dir))
            
            # 保存检查点
            similarity_results = {
                'success': True,
                'total_files': report['total_files'],
                'unique_files': report['unique_files'],
                'similar_groups': report['similar_groups'],
                'exact_duplicate_groups': report['exact_duplicate_groups'],
                'processing_time_seconds': report['processing_time_seconds'],
                'files_per_second': report['files_per_second'],
                'total_exact_duplicates': report['total_exact_duplicates'],
                'total_similar_files': report['total_similar_files'],
                'output_dir': str(similarity_output_dir)
            }
            
            self.save_checkpoint({
                'similarity_completed': True,
                'similarity_results': similarity_results
            })
            
            _get_logger().info(f"相似度分析完成！")
            _get_logger().info(f"总文件数: {report['total_files']}")
            _get_logger().info(f"唯一文件数: {report['unique_files']}")
            _get_logger().info(f"相似文件组: {report['similar_groups']}")
            _get_logger().info(f"处理时间: {report['processing_time_seconds']:.2f} 秒")
            _get_logger().info(f"结果保存到: {similarity_output_dir}")
            
            return similarity_results
            
        except Exception as e:
            _get_logger().error(f"相似度分析失败: {e}")
            return {
                'success': False,
                'error': str(e),
                'total_files': 0,
                'unique_files': 0,
                'similar_groups': 0,
                'exact_duplicate_groups': 0,
                'processing_time_seconds': 0,
                'output_dir': None
            }

    def deobfuscate_files(self, max_workers: int = 4, resume: bool = False) -> Dict[str, Any]:
        """反混淆JavaScript文件"""
        _get_logger().info("=" * 60)
        _get_logger().info("开始JavaScript文件反混淆")
        _get_logger().info("=" * 60)
        
        try:
            # 检查是否需要恢复
            if resume and 'deobfuscation_completed' in self.checkpoint_data:
                _get_logger().info("反混淆已完成，跳过此步骤")
                return self.checkpoint_data.get('deobfuscation_results', {})
            
            # 执行反混淆
            results = self.deobfuscator.process_all_files(self.dirs, max_workers)
            
            # 保存检查点
            self.save_checkpoint({
                'deobfuscation_completed': True,
                'deobfuscation_results': results
            })
            
            # 输出统计信息
            total = results['total']
            _get_logger().info(f"反混淆完成:")
            _get_logger().info(f"  - 总文件数: {total['total_files']}")
            _get_logger().info(f"  - 处理成功: {total['processed_files']}")
            _get_logger().info(f"  - 直接复制: {total['skipped_files']}")
            _get_logger().info(f"  - 处理失败: {total['failed_files']}")
            _get_logger().info(f"  - 成功率: {total.get('success_rate', 0):.1f}%")
            
            return results
            
        except Exception as e:
            _get_logger().error(f"反混淆失败: {e}")
            return {
                'static': {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0},
                'dynamic': {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0},
                'total': {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0, 'success_rate': 0}
            }
    
    def generate_final_report(self, static_results: Dict[str, Any], 
                            dynamic_results: Dict[str, Any], 
                            deobfuscation_results: Dict[str, Any],
                            similarity_results: Dict[str, Any] = None) -> Dict[str, Any]:
        """生成最终统计报告"""
        end_time = time.time()
        total_time = end_time - self.start_time
        
        # 计算总体统计
        total_discovered = static_results.get('total_discovered', 0) + dynamic_results.get('total_discovered', 0)
        total_downloaded = static_results.get('successful_downloads', 0) + dynamic_results.get('successful_downloads', 0)
        total_failed = static_results.get('failed_downloads', 0) + dynamic_results.get('failed_downloads', 0)
        
        # 计算文件大小
        static_files = static_results.get('downloaded_files', [])
        dynamic_files = dynamic_results.get('downloaded_files', [])
        total_size = sum(f.get('size', 0) for f in static_files + dynamic_files)
        
        # 反混淆统计
        deob_total = deobfuscation_results['total']
        
        # 判断整体是否成功
        success = total_downloaded > 0 or deob_total['processed_files'] > 0
        
        report = {
            'success': success,
            'total_files': total_downloaded,
            'output_dir': self.dirs['target_output_dir'],
            'error': None if success else '未发现任何JavaScript文件',
            'summary': {
                'start_time': datetime.fromtimestamp(self.start_time).strftime('%Y-%m-%d %H:%M:%S'),
                'end_time': datetime.fromtimestamp(end_time).strftime('%Y-%m-%d %H:%M:%S'),
                'total_time': f"{total_time:.2f}秒",
                'total_discovered': total_discovered,
                'total_downloaded': total_downloaded,
                'total_failed': total_failed,
                'download_success_rate': f"{(total_downloaded / total_discovered * 100) if total_discovered > 0 else 0:.1f}%",
                'total_size': format_file_size(total_size)
            },
            'static_crawling': {
                'discovered': static_results.get('total_discovered', 0),
                'downloaded': static_results.get('successful_downloads', 0),
                'failed': static_results.get('failed_downloads', 0),
                'pages_visited': static_results.get('visited_pages', 0)
            },
            'dynamic_crawling': {
                'discovered': dynamic_results.get('total_discovered', 0),
                'downloaded': dynamic_results.get('successful_downloads', 0),
                'failed': dynamic_results.get('failed_downloads', 0)
            },
            'deobfuscation': {
                'total_files': deob_total['total_files'],
                'processed': deob_total['processed_files'],
                'copied': deob_total['skipped_files'],
                'failed': deob_total['failed_files'],
                'success_rate': f"{deob_total.get('success_rate', 0):.1f}%"
            }
        }
        
        # 添加相似度分析结果（如果有）
        if similarity_results and similarity_results.get('success'):
            report['similarity_analysis'] = {
                'total_files': similarity_results.get('total_files', 0),
                'similar_groups': similarity_results.get('similar_groups', 0),
                'unique_files': similarity_results.get('unique_files', 0),
                'deduplication_rate': similarity_results.get('deduplication_rate', '0%'),
                'processing_time': similarity_results.get('processing_time', '0秒'),
                'output_dir': similarity_results.get('output_dir', '')
            }
        
        return report
    
    def _consolidate_detailed_reports(self):
        """整合所有详细报告到主输出目录"""
        try:
            from ..utils.report_generator import CrawlReportGenerator
            
            # 创建主报告生成器
            main_report_generator = CrawlReportGenerator(self.dirs['target_output_dir'])
            
            # 收集静态爬虫的报告数据
            if hasattr(self.static_crawler, 'report_generator'):
                static_generator = self.static_crawler.report_generator
                # 合并成功文件
                for file_info in static_generator.success_files:
                    main_report_generator.add_success_file(file_info)
                # 合并失败文件
                for file_info in static_generator.failed_files:
                    main_report_generator.add_failed_file(file_info)
                # 合并日志
                for log_entry in static_generator.detailed_logs:
                    main_report_generator.add_log(log_entry)
            
            # 收集动态爬虫的报告数据
            if hasattr(self.dynamic_crawler, 'report_generator'):
                dynamic_generator = self.dynamic_crawler.report_generator
                # 合并成功文件
                for file_info in dynamic_generator.success_files:
                    main_report_generator.add_success_file(file_info)
                # 合并失败文件
                for file_info in dynamic_generator.failed_files:
                    main_report_generator.add_failed_file(file_info)
                # 合并日志
                for log_entry in dynamic_generator.detailed_logs:
                    main_report_generator.add_log(log_entry)
            
            # 添加整合完成的日志
            main_report_generator.add_log("所有爬取报告已整合完成")
            
            # 保存整合后的报告
            main_report_generator.save_all_reports()
            
            # 打印整合报告摘要
            summary = main_report_generator.generate_summary_report()
            _get_logger().info(f"整合报告摘要: {summary['crawl_summary']}")
            
        except Exception as e:
            _get_logger().error(f"整合详细报告失败: {e}")
    
    def run(self, url: str, max_depth: int = 2, wait_time: int = 10, 
            max_workers: int = 4, playwright_tabs: int = 4, headless: bool = True, 
            mode: str = 'all', resume: bool = False, similarity_enabled: bool = True,
            similarity_threshold: float = 0.8, similarity_workers: int = None,
            auto_similarity: bool = True) -> Dict[str, Any]:
        """运行完整的爬取和反混淆流程"""
        _get_logger().info("JavaScript文件爬取和反混淆工具启动")
        _get_logger().info(f"目标URL: {url}")
        _get_logger().info(f"最大深度: {max_depth}")
        _get_logger().info(f"动态等待时间: {wait_time}秒")
        _get_logger().info(f"并行工作线程: {max_workers}")
        _get_logger().info(f"Playwright标签页数: {playwright_tabs}")
        _get_logger().info(f"无头模式: {headless}")
        _get_logger().info(f"爬取模式: {mode}")
        
        try:
            # 确保目录存在
            ensure_directories(self.target_url)
            
            # 加载检查点（如果需要恢复）
            if resume:
                checkpoint = self.load_checkpoint()
                if checkpoint:
                    self.checkpoint_data = checkpoint
            
            static_results = {}
            dynamic_results = {}
            
            # 根据模式选择执行的爬取类型
            if mode in ['static', 'all']:
                # 步骤1: 静态爬取
                static_results = self.crawl_static_js(url, max_depth, max_workers, resume)
            
            if mode in ['dynamic', 'all']:
                # 检查静态爬取是否因反爬虫机制失败（仅在全模式下检查）
                static_failed_anti_crawler = False
                if mode == 'all':
                    static_failed_anti_crawler = (
                        'static_failed_anti_crawler' in self.checkpoint_data or
                        self._is_anti_crawler_detected(static_results)
                    )
                
                # 步骤2: 动态爬取
                # 如果静态爬取失败，强制执行动态爬取
                if static_failed_anti_crawler:
                    _get_logger().info("由于检测到反爬虫机制，将强制执行动态爬取")
                    dynamic_results = self.crawl_dynamic_js(url, wait_time, playwright_tabs, headless, resume=False)  # 不跳过动态爬取
                else:
                    dynamic_results = self.crawl_dynamic_js(url, wait_time, playwright_tabs, headless, resume)
            
            # 步骤3: 反混淆（如果有文件需要处理）
            deobfuscation_results = {}
            if static_results or dynamic_results:
                deobfuscation_results = self.deobfuscate_files(max_workers, resume)
                _get_logger().info(f"反混淆完成，结果: {deobfuscation_results}")
            
            # 步骤4: 智能相似度分析（如果启用且有反编译文件）
            similarity_results = {}
            # 检查是否有文件需要分析
            total_files = deobfuscation_results.get('total', {}).get('total_files', 0)
            _get_logger().info(f"相似度分析检查: similarity_enabled={similarity_enabled}, auto_similarity={auto_similarity}, total_files={total_files}")
            _get_logger().info(f"deobfuscation_results结构: {deobfuscation_results}")
            if similarity_enabled or (auto_similarity and total_files > 0):
                _get_logger().info("开始执行相似度分析...")
                similarity_results = self.run_similarity_analysis(
                    similarity_threshold, similarity_workers, resume
                )
            else:
                _get_logger().info("跳过相似度分析")
            
            # 生成最终报告
            final_report = self.generate_final_report(
                static_results, dynamic_results, deobfuscation_results, similarity_results
            )
            
            # 整合所有详细报告
            self._consolidate_detailed_reports()
            
            # 输出最终报告
            self.print_final_report(final_report)
            
            # 清理检查点文件
            if self.checkpoint_file.exists():
                self.checkpoint_file.unlink()
                _get_logger().info("已清理检查点文件")
            
            return final_report
            
        except KeyboardInterrupt:
            _get_logger().warning("用户中断操作，已保存检查点")
            return {
                'success': False,
                'error': '用户中断操作',
                'total_files': 0,
                'output_dir': self.dirs.get('target_output_dir', '')
            }
        except Exception as e:
            _get_logger().error(f"程序执行失败: {e}")
            return {
                'success': False,
                'error': str(e),
                'total_files': 0,
                'output_dir': self.dirs.get('target_output_dir', '')
            }
    
    def print_final_report(self, report: Dict[str, Any]):
        """打印最终报告"""
        _get_logger().info("=" * 80)
        _get_logger().info("最终统计报告")
        _get_logger().info("=" * 80)
        
        summary = report['summary']
        _get_logger().info(f"开始时间: {summary['start_time']}")
        _get_logger().info(f"结束时间: {summary['end_time']}")
        _get_logger().info(f"总耗时: {summary['total_time']}")
        _get_logger().info("")
        
        _get_logger().info("文件发现和下载:")
        _get_logger().info(f"  总发现文件数: {summary['total_discovered']}")
        _get_logger().info(f"  成功下载数: {summary['total_downloaded']}")
        _get_logger().info(f"  下载失败数: {summary['total_failed']}")
        _get_logger().info(f"  下载成功率: {summary['download_success_rate']}")
        _get_logger().info(f"  总文件大小: {summary['total_size']}")
        _get_logger().info("")
        
        static = report['static_crawling']
        _get_logger().info("静态爬取:")
        _get_logger().info(f"  发现文件: {static['discovered']}")
        _get_logger().info(f"  下载成功: {static['downloaded']}")
        _get_logger().info(f"  下载失败: {static['failed']}")
        _get_logger().info(f"  访问页面: {static['pages_visited']}")
        _get_logger().info("")
        
        dynamic = report['dynamic_crawling']
        _get_logger().info("动态爬取:")
        _get_logger().info(f"  发现文件: {dynamic['discovered']}")
        _get_logger().info(f"  下载成功: {dynamic['downloaded']}")
        _get_logger().info(f"  下载失败: {dynamic['failed']}")
        _get_logger().info("")
        
        deob = report['deobfuscation']
        _get_logger().info("反混淆处理:")
        _get_logger().info(f"  总文件数: {deob['total_files']}")
        _get_logger().info(f"  反混淆处理: {deob['processed']}")
        _get_logger().info(f"  直接复制: {deob['copied']}")
        _get_logger().info(f"  处理失败: {deob['failed']}")
        _get_logger().info(f"  处理成功率: {deob['success_rate']}")
        _get_logger().info("")
        
        # 显示相似度分析结果（如果有）
        if 'similarity_analysis' in report:
            sim = report['similarity_analysis']
            _get_logger().info("智能相似度分析:")
            _get_logger().info(f"  分析文件数: {sim['total_files']}")
            _get_logger().info(f"  相似文件组: {sim['similar_groups']}")
            _get_logger().info(f"  唯一文件数: {sim['unique_files']}")
            _get_logger().info(f"  去重率: {sim['deduplication_rate']}")
            _get_logger().info(f"  处理时间: {sim['processing_time']}")
            _get_logger().info(f"  输出目录: {sim['output_dir']}")
            _get_logger().info("")
        
        _get_logger().info("=" * 80)

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='JavaScript文件爬取和反混淆工具')
    parser.add_argument('-u', '--url', dest='url', required=True, help='目标网站URL')
    parser.add_argument('-d', '--depth', type=int, default=2, help='爬取深度 (默认: 2)')
    parser.add_argument('-w', '--wait', type=int, default=3, help='页面等待时间(秒) (默认: 3)')
    parser.add_argument('-t', '--threads', type=int, default=2, help='静态爬取并行线程数 (默认: 2)')
    parser.add_argument('--playwright-tabs', type=int, default=4, help='Playwright同时打开的标签页数量 (默认: 4)')
    parser.add_argument('--headless', action='store_true', default=True, help='Playwright无头模式运行 (默认: True)')
    parser.add_argument('--no-headless', dest='headless', action='store_false', help='Playwright有头模式运行')
    parser.add_argument('--mode', choices=['static', 'dynamic', 'all'], default='all', 
                       help='爬取模式: static(仅静态), dynamic(仅动态), all(全部) (默认: all)')
    parser.add_argument('-r', '--resume', action='store_true', help='从检查点恢复')
    
    # 相似度检测相关参数
    parser.add_argument('--similarity', action='store_true', default=True, help='启用智能相似度检测和去重 (默认: True)')
    parser.add_argument('--similarity-threshold', type=float, default=0.8, 
                       help='相似度阈值 (0.0-1.0，默认: 0.8)')
    parser.add_argument('--similarity-workers', type=int, default=None,
                       help='相似度分析并行进程数 (默认: 自动检测)')
    parser.add_argument('--no-similarity', dest='similarity', action='store_false', 
                       help='禁用智能相似度检测和去重')
    parser.add_argument('--no-similarity-auto', action='store_true', 
                       help='禁用反编译后自动运行相似度分析')
    
    args = parser.parse_args()
    
    # 验证URL
    if not args.url.startswith(('http://', 'https://')):
        _get_logger().error("URL必须以http://或https://开头")
        sys.exit(1)
    
    # 创建爬取器并运行
    crawler = JSCrawler(args.url)
    try:
        result = crawler.crawl(
            depth=args.depth,
            wait_time=args.wait,
            max_workers=args.threads,
            playwright_tabs=getattr(args, 'playwright_tabs'),
            headless=args.headless,
            mode=args.mode,
            resume=args.resume,
            similarity_enabled=args.similarity,
            similarity_threshold=args.similarity_threshold,
            similarity_workers=args.similarity_workers,
            auto_similarity=not args.no_similarity_auto
        )
        
        # 输出结果
        if result.get('success'):
            print(f"\n爬取完成！总共处理了 {result.get('total_files', 0)} 个文件")
            print(f"📁 输出目录: {result.get('output_dir', crawler.dirs['target_output_dir'])}")
        else:
            print(f"\n爬取失败: {result.get('error', '未知错误')}")
            sys.exit(1)
    except Exception as e:
        _get_logger().error(f"程序执行失败: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()