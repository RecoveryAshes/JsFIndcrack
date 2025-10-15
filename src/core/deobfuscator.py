"""
JavaScript反混淆模块
"""
import os
import subprocess
import shutil
from pathlib import Path
from typing import Dict, Any, List, Optional
from concurrent.futures import ThreadPoolExecutor, as_completed
from tqdm import tqdm

from .config import (
    WEBCRACK_COMMAND, WEBCRACK_TIMEOUT, SUPPORTED_JS_EXTENSIONS
)
from ..utils.utils import is_minified_js, convert_to_utf8, format_file_size
from ..utils.logger import get_logger

logger = get_logger("deobfuscator")

class JSDeobfuscator:
    """JavaScript反混淆器"""
    
    def __init__(self):
        self.processed_files: List[Dict[str, Any]] = []
        self.failed_files: List[Dict[str, Any]] = []
        self.skipped_files: List[Dict[str, Any]] = []
        
    def check_webcrack_available(self) -> bool:
        """检查webcrack工具是否可用"""
        try:
            result = subprocess.run(
                [WEBCRACK_COMMAND, '--version'], 
                capture_output=True, 
                text=True, 
                timeout=10
            )
            if result.returncode == 0:
                logger.info(f"webcrack工具可用: {result.stdout.strip()}")
                return True
            else:
                logger.warning("webcrack工具不可用，将跳过反混淆步骤")
                return False
        except FileNotFoundError:
            logger.warning("未找到webcrack工具，请确保已安装并在PATH中")
            return False
        except Exception as e:
            logger.error(f"检查webcrack工具失败: {e}")
            return False
    
    def _should_deobfuscate(self, file_path: Path) -> bool:
        """判断文件是否需要反混淆"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # 检查文件大小
            if len(content) < 100:
                logger.debug(f"文件太小，跳过反混淆: {file_path}")
                return False
            
            # 检查是否为压缩/混淆代码
            if is_minified_js(content):
                logger.debug(f"检测到混淆代码，需要反混淆: {file_path}")
                return True
            
            logger.debug(f"文件未混淆，跳过反混淆: {file_path}")
            return False
            
        except Exception as e:
            logger.error(f"检查文件是否需要反混淆失败 {file_path}: {e}")
            return False
    
    def _deobfuscate_file(self, input_path: Path, output_path: Path) -> bool:
        """使用webcrack反混淆单个文件"""
        try:
            logger.info(f"正在反混淆: {input_path}")
            
            # 确保输出目录存在
            output_path.parent.mkdir(parents=True, exist_ok=True)
            
            # webcrack会创建一个临时目录，我们需要指定这个目录
            temp_output_dir = output_path.parent / f"temp_{input_path.stem}"
            
            # 构建webcrack命令
            cmd = [
                WEBCRACK_COMMAND,
                str(input_path),
                '--output', str(temp_output_dir)
            ]
            
            # 执行webcrack命令
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=WEBCRACK_TIMEOUT
            )
            
            if result.returncode == 0:
                # webcrack创建的实际文件路径
                webcrack_output = temp_output_dir / "deobfuscated.js"
                
                if webcrack_output.exists():
                    # 将webcrack的输出移动到我们期望的位置
                    shutil.move(str(webcrack_output), str(output_path))
                    
                    # 清理临时目录
                    if temp_output_dir.exists():
                        shutil.rmtree(temp_output_dir)
                    
                    # 确保输出文件为UTF-8编码
                    convert_to_utf8(output_path)
                    
                    # 记录成功处理
                    file_info = {
                        'input_path': str(input_path),
                        'output_path': str(output_path),
                        'original_size': input_path.stat().st_size,
                        'deobfuscated_size': output_path.stat().st_size,
                        'status': 'success'
                    }
                    self.processed_files.append(file_info)
                    
                    logger.info(f"反混淆成功: {input_path} -> {output_path}")
                    return True
                else:
                    # 清理临时目录
                    if temp_output_dir.exists():
                        shutil.rmtree(temp_output_dir)
                    logger.error(f"webcrack执行成功但未生成输出文件: {webcrack_output}")
                    return False
            else:
                # 清理临时目录
                if temp_output_dir.exists():
                    shutil.rmtree(temp_output_dir)
                    
                error_msg = result.stderr.strip() if result.stderr else "未知错误"
                logger.error(f"webcrack执行失败: {error_msg}")
                
                # 记录失败信息
                self.failed_files.append({
                    'input_path': str(input_path),
                    'error': error_msg,
                    'status': 'failed'
                })
                return False
                
        except subprocess.TimeoutExpired:
            # 清理临时目录
            if 'temp_output_dir' in locals() and temp_output_dir.exists():
                shutil.rmtree(temp_output_dir)
                
            logger.error(f"webcrack执行超时: {input_path}")
            self.failed_files.append({
                'input_path': str(input_path),
                'error': '执行超时',
                'status': 'timeout'
            })
            return False
            
        except Exception as e:
            # 清理临时目录
            if 'temp_output_dir' in locals() and temp_output_dir.exists():
                shutil.rmtree(temp_output_dir)
                
            logger.error(f"反混淆文件失败 {input_path}: {e}")
            self.failed_files.append({
                'input_path': str(input_path),
                'error': str(e),
                'status': 'error'
            })
            return False
    
    def _copy_unobfuscated_file(self, input_path: Path, output_path: Path) -> bool:
        """复制未混淆的文件到输出目录"""
        try:
            # 确保输出目录存在
            output_path.parent.mkdir(parents=True, exist_ok=True)
            
            # 复制文件
            shutil.copy2(input_path, output_path)
            
            # 确保UTF-8编码
            convert_to_utf8(output_path)
            
            # 记录跳过的文件
            file_info = {
                'input_path': str(input_path),
                'output_path': str(output_path),
                'size': input_path.stat().st_size,
                'status': 'copied',
                'reason': 'not_obfuscated'
            }
            self.skipped_files.append(file_info)
            
            logger.debug(f"复制未混淆文件: {input_path} -> {output_path}")
            return True
            
        except Exception as e:
            logger.error(f"复制文件失败 {input_path}: {e}")
            return False
    
    def process_file(self, input_path: Path, file_type: str, dirs: Dict[str, Path] = None) -> bool:
        """处理单个JavaScript文件"""
        try:
            # 如果没有提供dirs，使用默认路径
            if dirs is None:
                original_dir = ORIGINAL_DIR
                decrypted_dir = DECRYPTED_DIR
            else:
                original_dir = dirs['original_dir']
                decrypted_dir = dirs['decrypted_dir']
            
            # 生成输出路径 - 在新的目录结构中，直接使用文件名
            relative_path = input_path.relative_to(original_dir)
            output_path = decrypted_dir / relative_path
            
            # 检查是否需要反混淆
            if self._should_deobfuscate(input_path):
                # 检查webcrack是否可用
                if self.check_webcrack_available():
                    return self._deobfuscate_file(input_path, output_path)
                else:
                    # webcrack不可用，直接复制文件
                    return self._copy_unobfuscated_file(input_path, output_path)
            else:
                # 文件未混淆，直接复制
                return self._copy_unobfuscated_file(input_path, output_path)
                
        except Exception as e:
            logger.error(f"处理文件失败 {input_path}: {e}")
            return False
    
    def process_directory(self, file_type: str, source_dir: Path, max_workers: int = 4, dirs: Dict[str, Path] = None) -> Dict[str, Any]:
        """处理指定类型的目录中的所有JS文件"""
        input_dir = source_dir
        
        if not input_dir.exists():
            logger.warning(f"输入目录不存在: {input_dir}")
            return {
                'total_files': 0,
                'processed_files': 0,
                'failed_files': 0,
                'skipped_files': 0
            }
        
        # 查找所有JavaScript文件
        js_files = []
        for ext in ['.js', '.mjs', '.jsx']:
            js_files.extend(input_dir.rglob(f'*{ext}'))
        
        if not js_files:
            logger.info(f"在 {input_dir} 中未找到JavaScript文件")
            return {
                'total_files': 0,
                'processed_files': 0,
                'failed_files': 0,
                'skipped_files': 0
            }
        
        logger.info(f"开始处理 {len(js_files)} 个{file_type}JavaScript文件...")
        
        # 使用线程池并行处理文件，添加进度条
        with tqdm(total=len(js_files), desc=f"处理{file_type}JS文件", unit="文件") as pbar:
            with ThreadPoolExecutor(max_workers=max_workers) as executor:
                future_to_file = {
                    executor.submit(self.process_file, file_path, file_type, dirs): file_path
                    for file_path in js_files
                }
                
                for future in as_completed(future_to_file):
                    file_path = future_to_file[future]
                    try:
                        success = future.result()
                        pbar.set_postfix({
                            '成功': len(self.processed_files),
                            '跳过': len(self.skipped_files),
                            '失败': len(self.failed_files)
                        })
                        pbar.update(1)
                        
                        if success:
                            logger.debug(f"处理成功: {file_path}")
                        else:
                            logger.warning(f"处理失败: {file_path}")
                    except Exception as e:
                        logger.error(f"处理文件异常 {file_path}: {e}")
                        pbar.update(1)
        
        return {
            'total_files': len(js_files),
            'processed_files': len(self.processed_files),
            'failed_files': len(self.failed_files),
            'skipped_files': len(self.skipped_files)
        }
    
    def process_all_files(self, dirs: Dict[str, Path], max_workers: int = 4) -> Dict[str, Any]:
        """处理所有JavaScript文件（静态和动态）"""
        logger.info("开始反混淆处理...")
        
        # 重置统计信息
        self.processed_files.clear()
        self.failed_files.clear()
        self.skipped_files.clear()
        
        # 在新的目录结构中，所有文件都在同一个"编译前"目录中
        original_dir = dirs['original_dir']
        if original_dir.exists():
            # 处理所有文件
            all_stats = self.process_directory('', original_dir, max_workers, dirs)
            static_stats = all_stats  # 为了保持兼容性
            dynamic_stats = {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0}
        else:
            static_stats = {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0}
            dynamic_stats = {'total_files': 0, 'processed_files': 0, 'failed_files': 0, 'skipped_files': 0}
        
        # 合并统计信息
        total_stats = {
            'static': static_stats,
            'dynamic': dynamic_stats,
            'total': {
                'total_files': static_stats['total_files'] + dynamic_stats['total_files'],
                'processed_files': static_stats['processed_files'] + dynamic_stats['processed_files'],
                'failed_files': static_stats['failed_files'] + dynamic_stats['failed_files'],
                'skipped_files': static_stats['skipped_files'] + dynamic_stats['skipped_files']
            }
        }
        
        # 计算成功率
        total_files = total_stats['total']['total_files']
        if total_files > 0:
            success_rate = (total_stats['total']['processed_files'] + 
                          total_stats['total']['skipped_files']) / total_files * 100
            total_stats['total']['success_rate'] = success_rate
        else:
            total_stats['total']['success_rate'] = 0
        
        logger.info(f"反混淆处理完成: 总文件数={total_files}, "
                   f"成功={total_stats['total']['processed_files']}, "
                   f"跳过={total_stats['total']['skipped_files']}, "
                   f"失败={total_stats['total']['failed_files']}")
        
        return total_stats
    
    def get_detailed_report(self) -> Dict[str, Any]:
        """获取详细的处理报告"""
        total_original_size = sum(
            file_info.get('original_size', 0) for file_info in self.processed_files
        )
        total_deobfuscated_size = sum(
            file_info.get('deobfuscated_size', 0) for file_info in self.processed_files
        )
        total_copied_size = sum(
            file_info.get('size', 0) for file_info in self.skipped_files
        )
        
        return {
            'processed_files': self.processed_files,
            'failed_files': self.failed_files,
            'skipped_files': self.skipped_files,
            'statistics': {
                'total_processed': len(self.processed_files),
                'total_failed': len(self.failed_files),
                'total_skipped': len(self.skipped_files),
                'original_size': format_file_size(total_original_size),
                'deobfuscated_size': format_file_size(total_deobfuscated_size),
                'copied_size': format_file_size(total_copied_size),
                'total_output_size': format_file_size(total_deobfuscated_size + total_copied_size)
            }
        }