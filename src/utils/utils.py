"""
工具函数模块
"""
import os
import re
import json
import hashlib
import chardet
from pathlib import Path
from urllib.parse import urljoin, urlparse, unquote
from typing import Optional, Dict, Any
from .logger import get_logger

logger = get_logger("utils")

def is_valid_url(url: str) -> bool:
    """检查URL是否有效"""
    try:
        result = urlparse(url)
        return all([result.scheme, result.netloc])
    except Exception:
        return False

def normalize_url(base_url: str, url: str) -> str:
    """标准化URL"""
    if url.startswith('//'):
        # 协议相对URL
        parsed_base = urlparse(base_url)
        return f"{parsed_base.scheme}:{url}"
    elif url.startswith('/'):
        # 绝对路径
        return urljoin(base_url, url)
    elif url.startswith('http'):
        # 完整URL
        return url
    else:
        # 相对路径
        return urljoin(base_url, url)

def is_javascript_file(url: str) -> bool:
    """判断URL是否指向JavaScript文件"""
    from ..core.config import SUPPORTED_JS_EXTENSIONS
    
    # 移除查询参数和片段
    parsed = urlparse(url)
    path = parsed.path.lower()
    
    # 检查文件扩展名
    for ext in SUPPORTED_JS_EXTENSIONS:
        if path.endswith(ext):
            return True
    
    # 检查Content-Type（如果可用）
    if 'javascript' in path or 'js' in path:
        return True
    
    return False

def generate_file_path(url: str, target_url: str, file_type: str = "static") -> Path:
    """
    根据URL生成文件保存路径
    
    Args:
        url: JavaScript文件的URL
        target_url: 目标网站的URL
        file_type: 文件类型，'static' 或 'dynamic'
    
    Returns:
        Path: 文件保存路径
    """
    from ..core.config import get_directory_structure
    
    # 获取目录结构
    dirs = get_directory_structure(target_url)
    
    # 解析URL获取文件名
    parsed_url = urlparse(url)
    filename = Path(parsed_url.path).name
    
    # 如果没有文件名，生成一个
    if not filename or not filename.endswith('.js'):
        filename = f"{calculate_file_hash(url)}.js"
    
    # 清理文件名
    filename = sanitize_filename(filename)
    
    # 确定保存目录
    if file_type == "static":
        save_dir = dirs['static_original_dir']
    else:
        save_dir = dirs['dynamic_original_dir']
    
    return save_dir / filename

def calculate_file_hash(file_path: Path) -> str:
    """计算文件MD5哈希值"""
    hash_md5 = hashlib.md5()
    try:
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(4096), b""):
                hash_md5.update(chunk)
        return hash_md5.hexdigest()
    except Exception as e:
        logger.error(f"计算文件哈希失败 {file_path}: {e}")
        return ""

def detect_encoding(file_path: Path) -> str:
    """检测文件编码"""
    try:
        with open(file_path, 'rb') as f:
            raw_data = f.read()
            result = chardet.detect(raw_data)
            return result['encoding'] or 'utf-8'
    except Exception as e:
        logger.error(f"检测文件编码失败 {file_path}: {e}")
        return 'utf-8'

def convert_to_utf8(file_path: Path) -> bool:
    """将文件转换为UTF-8编码"""
    try:
        # 检测原始编码
        original_encoding = detect_encoding(file_path)
        
        if original_encoding.lower() == 'utf-8':
            return True
        
        # 读取原始内容
        with open(file_path, 'r', encoding=original_encoding) as f:
            content = f.read()
        
        # 写入UTF-8格式
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        logger.info(f"文件编码转换成功: {file_path} ({original_encoding} -> UTF-8)")
        return True
        
    except Exception as e:
        logger.error(f"文件编码转换失败 {file_path}: {e}")
        return False

def save_checkpoint(data: Dict[Any, Any], checkpoint_file: Path) -> bool:
    """保存检查点数据"""
    try:
        with open(checkpoint_file, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
        return True
    except Exception as e:
        logger.error(f"保存检查点失败: {e}")
        return False

def load_checkpoint(checkpoint_file: Path) -> Optional[Dict[Any, Any]]:
    """加载检查点数据"""
    try:
        if checkpoint_file.exists():
            with open(checkpoint_file, 'r', encoding='utf-8') as f:
                return json.load(f)
    except Exception as e:
        logger.error(f"加载检查点失败: {e}")
    return None

def sanitize_filename(filename: str) -> str:
    """清理文件名，移除非法字符"""
    # 移除或替换非法字符
    filename = re.sub(r'[<>:"/\\|?*]', '_', filename)
    # 移除控制字符
    filename = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', filename)
    # 限制长度
    if len(filename) > 255:
        name, ext = os.path.splitext(filename)
        filename = name[:255-len(ext)] + ext
    
    return filename

def format_file_size(size_bytes: int) -> str:
    """格式化文件大小显示"""
    if size_bytes == 0:
        return "0B"
    
    size_names = ["B", "KB", "MB", "GB"]
    i = 0
    while size_bytes >= 1024 and i < len(size_names) - 1:
        size_bytes /= 1024.0
        i += 1
    
    return f"{size_bytes:.1f}{size_names[i]}"

def is_minified_js(content: str) -> bool:
    """判断JavaScript代码是否被压缩/混淆"""
    lines = content.split('\n')
    
    # 检查平均行长度
    if lines:
        avg_line_length = sum(len(line) for line in lines) / len(lines)
        if avg_line_length > 200:  # 平均行长度超过200字符
            return True
    
    # 检查是否包含典型的混淆特征
    minified_patterns = [
        r'[a-zA-Z]\$[a-zA-Z]',  # 变量名包含$
        r'_0x[a-f0-9]+',        # 十六进制变量名
        r'[a-zA-Z]{1,2}\[\d+\]', # 短变量名数组访问
    ]
    
    for pattern in minified_patterns:
        if re.search(pattern, content):
            return True
    
    return False