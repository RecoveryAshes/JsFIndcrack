"""
爬取结果报告生成器
"""
import json
import os
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional

from ..core.config import REPORT_CONFIG


class CrawlReportGenerator:
    """爬取结果报告生成器"""
    
    def __init__(self, target_dir: Path):
        """
        初始化报告生成器
        
        Args:
            target_dir: 目标输出目录
        """
        self.target_dir = Path(target_dir)
        self.start_time = datetime.now()
        self.success_files = []
        self.failed_files = []
        self.detailed_logs = []
        
    def add_success_file(self, file_info: Dict[str, Any]):
        """
        添加成功下载的文件信息
        
        Args:
            file_info: 文件信息字典，包含url, file_path, size, download_time等
        """
        file_info['timestamp'] = datetime.now().isoformat()
        self.success_files.append(file_info)
        self._add_detailed_log(f"SUCCESS: {file_info.get('url', 'Unknown')} -> {file_info.get('file_path', 'Unknown')}")
        
    def add_failed_file(self, file_info: Dict[str, Any]):
        """
        添加失败的文件信息
        
        Args:
            file_info: 文件信息字典，包含url, error, attempt_time等
        """
        file_info['timestamp'] = datetime.now().isoformat()
        self.failed_files.append(file_info)
        self._add_detailed_log(f"FAILED: {file_info.get('url', 'Unknown')} - {file_info.get('error', 'Unknown error')}")
        
    def add_log(self, message: str, level: str = "INFO"):
        """
        添加详细日志
        
        Args:
            message: 日志消息
            level: 日志级别
        """
        self._add_detailed_log(f"{level}: {message}")
        
    def _add_detailed_log(self, message: str):
        """添加详细日志条目"""
        timestamp = datetime.now().isoformat()
        self.detailed_logs.append(f"[{timestamp}] {message}")
        
    def generate_summary_report(self) -> Dict[str, Any]:
        """生成摘要报告"""
        end_time = datetime.now()
        duration = (end_time - self.start_time).total_seconds()
        
        total_files = len(self.success_files) + len(self.failed_files)
        success_rate = (len(self.success_files) / total_files * 100) if total_files > 0 else 0
        
        # 计算总下载大小
        total_size = sum(file_info.get('size', 0) for file_info in self.success_files)
        
        # 统计JS和MAP文件
        js_files = 0
        map_files = 0
        js_size = 0
        map_size = 0
        
        for file_info in self.success_files:
            file_path = file_info.get('file_path', '')
            url = file_info.get('url', '')
            size = file_info.get('size', 0)
            
            if file_path:
                if file_path.lower().endswith('.js'):
                    js_files += 1
                    js_size += size
                elif file_path.lower().endswith('.map') or url.endswith('.js.map'):
                    map_files += 1
                    map_size += size
        
        return {
            'crawl_summary': {
                'start_time': self.start_time.isoformat(),
                'end_time': end_time.isoformat(),
                'duration_seconds': duration,
                'total_files_attempted': total_files,
                'successful_downloads': len(self.success_files),
                'failed_downloads': len(self.failed_files),
                'success_rate_percent': round(success_rate, 2),
                'total_size_bytes': total_size,
                'total_size_mb': round(total_size / (1024 * 1024), 2),
                'target_directory': str(self.target_dir),
                'javascript_files': js_files,
                'sourcemap_files': map_files,
                'javascript_size_bytes': js_size,
                'sourcemap_size_bytes': map_size,
                'javascript_size_mb': round(js_size / (1024 * 1024), 2),
                'sourcemap_size_mb': round(map_size / (1024 * 1024), 2)
            }
        }
        
    def generate_crawl_report(self) -> Dict[str, Any]:
        """生成完整的爬取报告"""
        summary = self.generate_summary_report()
        
        return {
            **summary,
            'successful_files': self.success_files,
            'failed_files': self.failed_files,
            'file_types_summary': self._get_file_types_summary(),
            'error_summary': self._get_error_summary()
        }
        
    def _get_file_types_summary(self) -> Dict[str, int]:
        """获取文件类型统计"""
        file_types = {}
        detailed_types = {}
        
        for file_info in self.success_files:
            file_path = file_info.get('file_path', '')
            url = file_info.get('url', '')
            
            if file_path:
                ext = Path(file_path).suffix.lower()
                file_types[ext] = file_types.get(ext, 0) + 1
                
                # 详细分类
                if ext == '.js':
                    if '.min.js' in file_path.lower():
                        detailed_types['JavaScript (Minified)'] = detailed_types.get('JavaScript (Minified)', 0) + 1
                    else:
                        detailed_types['JavaScript (Regular)'] = detailed_types.get('JavaScript (Regular)', 0) + 1
                elif ext == '.map':
                    detailed_types['Source Map'] = detailed_types.get('Source Map', 0) + 1
                elif url.endswith('.js.map'):
                    # 处理没有扩展名但URL表明是source map的情况
                    detailed_types['Source Map'] = detailed_types.get('Source Map', 0) + 1
                    file_types['.map'] = file_types.get('.map', 0) + 1
                else:
                    detailed_types[f'Other ({ext})'] = detailed_types.get(f'Other ({ext})', 0) + 1
        
        # 返回包含基本扩展名统计和详细分类的结果
        return {
            'by_extension': file_types,
            'by_type': detailed_types
        }
        
    def _get_error_summary(self) -> Dict[str, int]:
        """获取错误类型统计"""
        error_types = {}
        for file_info in self.failed_files:
            error = file_info.get('error', 'Unknown error')
            # 简化错误信息，提取主要错误类型
            if 'timeout' in error.lower():
                error_type = 'Timeout'
            elif 'connection' in error.lower():
                error_type = 'Connection Error'
            elif '404' in error:
                error_type = 'Not Found (404)'
            elif '403' in error:
                error_type = 'Forbidden (403)'
            elif '500' in error:
                error_type = 'Server Error (500)'
            elif 'ssl' in error.lower():
                error_type = 'SSL Error'
            else:
                error_type = 'Other Error'
                
            error_types[error_type] = error_types.get(error_type, 0) + 1
        return error_types
        
    def save_all_reports(self):
        """保存所有报告文件到目标目录"""
        # 确保目标目录存在
        self.target_dir.mkdir(parents=True, exist_ok=True)
        
        # 保存完整爬取报告
        crawl_report = self.generate_crawl_report()
        self._save_json_report(REPORT_CONFIG['crawl_report'], crawl_report)
        
        # 保存成功文件列表
        self._save_json_report(REPORT_CONFIG['success_report'], {
            'successful_files': self.success_files,
            'count': len(self.success_files)
        })
        
        # 保存失败文件列表
        self._save_json_report(REPORT_CONFIG['failed_report'], {
            'failed_files': self.failed_files,
            'count': len(self.failed_files)
        })
        
        # 保存摘要报告
        summary_report = self.generate_summary_report()
        self._save_json_report(REPORT_CONFIG['summary_report'], summary_report)
        
        # 保存详细日志
        self._save_detailed_log()
        
    def _save_json_report(self, filename: str, data: Dict[str, Any]):
        """保存JSON格式的报告"""
        file_path = self.target_dir / filename
        try:
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
        except Exception as e:
            print(f"保存报告文件失败 {filename}: {e}")
            
    def _save_detailed_log(self):
        """保存详细日志文件"""
        file_path = self.target_dir / REPORT_CONFIG['detailed_log']
        try:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write('\n'.join(self.detailed_logs))
        except Exception as e:
            print(f"保存详细日志失败: {e}")
            
    def print_summary(self):
        """打印爬取摘要到控制台"""
        summary = self.generate_summary_report()['crawl_summary']
        
        print("\n" + "="*60)
        print("爬取结果摘要")
        print("="*60)
        print(f"开始时间: {summary['start_time']}")
        print(f"结束时间: {summary['end_time']}")
        print(f"耗时: {summary['duration_seconds']:.2f} 秒")
        print(f"尝试下载文件总数: {summary['total_files_attempted']}")
        print(f"成功下载: {summary['successful_downloads']}")
        print(f"失败下载: {summary['failed_downloads']}")
        print(f"成功率: {summary['success_rate_percent']}%")
        print(f"总下载大小: {summary['total_size_mb']} MB")
        print("")
        print("文件类型统计:")
        print(f"  JavaScript文件: {summary['javascript_files']} 个 ({summary['javascript_size_mb']} MB)")
        print(f"  Source Map文件: {summary['sourcemap_files']} 个 ({summary['sourcemap_size_mb']} MB)")
        print("")
        print(f"目标目录: {summary['target_directory']}")
        print("="*60)