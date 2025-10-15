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
    
    # 检查是否明确包含JavaScript相关的路径段（更严格的检查）
    path_segments = path.split('/')
    filename = path_segments[-1] if path_segments else ''
    
    # 只有当文件名本身包含javascript或js，且不是其他文件类型时才认为是JS文件
    if filename:
        # 排除明显的非JS文件扩展名
        non_js_extensions = ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.css', '.html', '.htm', '.xml', '.json', '.txt']
        for non_ext in non_js_extensions:
            if filename.endswith(non_ext):
                return False
        
        # 检查文件名是否明确表示是JavaScript文件
        if (filename.endswith('.js') or 
            filename.endswith('.mjs') or 
            filename.endswith('.jsx') or
            'javascript' in filename):
            return True
    
    return False

def is_map_file(url: str) -> bool:
    """判断URL是否指向source map文件"""
    from ..core.config import SUPPORTED_MAP_EXTENSIONS
    
    # 移除查询参数和片段
    parsed = urlparse(url)
    path = parsed.path.lower()
    
    # 检查文件扩展名
    for ext in SUPPORTED_MAP_EXTENSIONS:
        if path.endswith(ext):
            return True
    
    return False

def is_supported_file(url: str) -> bool:
    """判断URL是否指向支持的文件类型（JS或MAP）"""
    return is_javascript_file(url) or is_map_file(url)

def get_file_type(url: str) -> str:
    """获取文件类型"""
    if is_javascript_file(url):
        return 'js'
    elif is_map_file(url):
        return 'map'
    else:
        return 'unknown'

def is_external_domain(js_url: str, target_url: str) -> bool:
    """
    判断JS文件URL是否来自外部域名
    
    Args:
        js_url: JavaScript文件的URL
        target_url: 目标网站的URL
    
    Returns:
        bool: 如果是外部域名返回True，否则返回False
    """
    try:
        js_parsed = urlparse(js_url)
        target_parsed = urlparse(target_url)
        
        # 获取域名（netloc）
        js_domain = js_parsed.netloc.lower()
        target_domain = target_parsed.netloc.lower()
        
        # 移除www前缀进行比较
        js_domain = re.sub(r'^www\.', '', js_domain)
        target_domain = re.sub(r'^www\.', '', target_domain)
        
        # 如果JS文件没有域名（相对路径），则认为是内部文件
        if not js_domain:
            return False
            
        # 比较域名是否不同
        return js_domain != target_domain
        
    except Exception as e:
        logger.warning(f"判断外部域名时出错: {e}")
        return False

def get_domain_from_url(url: str) -> str:
    """
    从URL中提取域名并清理
    
    Args:
        url: URL字符串
    
    Returns:
        str: 清理后的域名
    """
    try:
        parsed = urlparse(url)
        domain = parsed.netloc.lower()
        
        # 移除www前缀
        domain = re.sub(r'^www\.', '', domain)
        
        # 清理域名，移除不合法的文件名字符
        clean_domain = re.sub(r'[<>:"/\\|?*]', '_', domain)
        clean_domain = clean_domain.strip('.')
        
        # 如果域名为空，使用默认名称
        if not clean_domain:
            clean_domain = 'unknown_domain'
            
        return clean_domain
        
    except Exception as e:
        logger.warning(f"提取域名时出错: {e}")
        return 'unknown_domain'

def generate_unique_file_path(url: str, target_url: str, file_type: str = "static", existing_files: set = None) -> Path:
    """
    根据URL生成唯一的文件保存路径，避免文件名冲突
    支持为所有域名JS文件创建以域名命名的单独目录
    
    Args:
        url: JavaScript文件的URL
        target_url: 目标网站的URL
        file_type: 文件类型，'static' 或 'dynamic'
        existing_files: 已存在的文件名集合
    
    Returns:
        Path: 唯一的文件保存路径
    """
    from ..core.config import get_directory_structure
    
    # 获取目录结构
    dirs = get_directory_structure(target_url)
    
    # 解析URL获取文件名
    parsed_url = urlparse(url)
    filename = Path(parsed_url.path).name
    
    # 如果没有文件名，生成一个
    if not filename or not filename.endswith('.js'):
        filename = f"{calculate_url_hash(url)}.js"
    
    # 清理文件名
    filename = sanitize_filename(filename)
    
    # 确定基础保存目录
    if file_type == "static":
        base_save_dir = dirs['static_original_dir']
    else:
        base_save_dir = dirs['dynamic_original_dir']
    
    # 为所有JS文件创建以域名命名的子目录
    if is_external_domain(url, target_url):
        # 外部域名
        domain = get_domain_from_url(url)
    else:
        # 目标域名
        domain = get_domain_from_url(target_url)
    
    # 创建域名子目录
    save_dir = base_save_dir / domain
    
    # 确保域名目录存在
    save_dir.mkdir(parents=True, exist_ok=True)
    
    logger.info(f"JS文件将保存到域名目录: {save_dir}")
    
    # 处理文件名冲突
    base_path = save_dir / filename
    if existing_files is None:
        existing_files = set()
    
    # 如果文件名已存在，添加数字后缀
    if filename in existing_files or base_path.exists():
        name_part, ext_part = os.path.splitext(filename)
        counter = 1
        while True:
            new_filename = f"{name_part}_{counter}{ext_part}"
            new_path = save_dir / new_filename
            if new_filename not in existing_files and not new_path.exists():
                filename = new_filename
                break
            counter += 1
    
    return save_dir / filename

def generate_file_path(url: str, target_url: str, file_type: str = "static") -> Path:
    """
    根据URL生成文件保存路径（保持向后兼容）
    
    Args:
        url: JavaScript文件的URL
        target_url: 目标网站的URL
        file_type: 文件类型，'static' 或 'dynamic'
    
    Returns:
        Path: 文件保存路径
    """
    return generate_unique_file_path(url, target_url, file_type)

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

def calculate_content_hash(content: bytes) -> str:
    """计算内容MD5哈希值"""
    try:
        hash_md5 = hashlib.md5()
        hash_md5.update(content)
        return hash_md5.hexdigest()
    except Exception as e:
        logger.error(f"计算内容哈希失败: {e}")
        return ""

def calculate_url_hash(url: str) -> str:
    """计算URL的MD5哈希值，用于生成文件名"""
    try:
        hash_md5 = hashlib.md5()
        hash_md5.update(url.encode('utf-8'))
        return hash_md5.hexdigest()[:8]  # 只取前8位，避免文件名过长
    except Exception as e:
        logger.error(f"计算URL哈希失败 {url}: {e}")
        return "unknown"

def is_duplicate_content(content: bytes, existing_hashes: set) -> bool:
    """检查内容是否重复"""
    content_hash = calculate_content_hash(content)
    return content_hash in existing_hashes if content_hash else False

def get_content_hash(content: bytes) -> str:
    """计算内容MD5哈希值（别名函数）"""
    return calculate_content_hash(content)

def is_file_already_downloaded(url: str, target_url: str, file_type: str = "static") -> bool:
    """
    检查文件是否已经下载过（基于文件名）
    支持所有域名目录的文件检查
    
    Args:
        url: JavaScript文件的URL
        target_url: 目标网站的URL
        file_type: 文件类型，'static' 或 'dynamic'
    
    Returns:
        bool: 如果文件已存在返回True，否则返回False
    """
    try:
        from ..core.config import get_directory_structure
        
        # 获取目录结构
        dirs = get_directory_structure(target_url)
        
        # 解析URL获取文件名
        parsed_url = urlparse(url)
        filename = Path(parsed_url.path).name
        
        # 如果没有文件名，生成一个
        if not filename or not filename.endswith('.js'):
            filename = f"{calculate_url_hash(url)}.js"
        
        # 清理文件名
        filename = sanitize_filename(filename)
        
        # 确定基础保存目录
        if file_type == "static":
            base_save_dir = dirs['static_original_dir']
        else:
            base_save_dir = dirs['dynamic_original_dir']
        
        # 为所有JS文件确定域名子目录
        if is_external_domain(url, target_url):
            # 外部域名
            domain = get_domain_from_url(url)
        else:
            # 目标域名
            domain = get_domain_from_url(target_url)
        
        # 构建域名子目录路径
        save_dir = base_save_dir / domain
        
        # 检查文件是否存在
        file_path = save_dir / filename
        return file_path.exists()
        
    except Exception as e:
        logger.error(f"检查文件是否存在时出错: {e}")
        return False

def detect_encoding(file_path: Path) -> str:
    """检测文件编码，使用多种方法确保准确性"""
    try:
        with open(file_path, 'rb') as f:
            raw_data = f.read()
            
        # 首先尝试chardet检测
        result = chardet.detect(raw_data)
        detected_encoding = result['encoding']
        confidence = result['confidence']
        
        # 如果置信度太低，尝试常见编码
        if not detected_encoding or confidence < 0.7:
            common_encodings = ['utf-8', 'utf-8-sig', 'latin-1', 'cp1252', 'iso-8859-1']
            for encoding in common_encodings:
                try:
                    raw_data.decode(encoding)
                    logger.debug(f"使用回退编码 {encoding} 检测文件: {file_path}")
                    return encoding
                except UnicodeDecodeError:
                    continue
        
        return detected_encoding or 'utf-8'
        
    except Exception as e:
        logger.error(f"检测文件编码失败 {file_path}: {e}")
        return 'utf-8'

def convert_to_utf8(file_path: Path) -> bool:
    """将文件转换为UTF-8编码，使用多种编码回退机制"""
    try:
        # 检测原始编码
        original_encoding = detect_encoding(file_path)
        
        # 如果已经是UTF-8，检查文件是否可以正常读取
        if original_encoding.lower() in ['utf-8', 'utf-8-sig']:
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    f.read()
                return True
            except UnicodeDecodeError:
                # UTF-8检测错误，继续尝试其他编码
                pass
        
        # 尝试多种编码读取文件
        encodings_to_try = [original_encoding, 'utf-8', 'utf-8-sig', 'latin-1', 'cp1252', 'iso-8859-1']
        content = None
        successful_encoding = None
        
        for encoding in encodings_to_try:
            if not encoding:
                continue
            try:
                with open(file_path, 'r', encoding=encoding, errors='strict') as f:
                    content = f.read()
                successful_encoding = encoding
                break
            except (UnicodeDecodeError, LookupError):
                continue
        
        # 如果所有编码都失败，使用errors='replace'读取
        if content is None:
            try:
                with open(file_path, 'r', encoding='utf-8', errors='replace') as f:
                    content = f.read()
                successful_encoding = 'utf-8 (with errors replaced)'
                logger.warning(f"使用错误替换模式读取文件: {file_path}")
            except Exception as e:
                logger.error(f"无法读取文件 {file_path}: {e}")
                return False
        
        # 如果成功读取且不是UTF-8，则转换为UTF-8
        if successful_encoding and not successful_encoding.lower().startswith('utf-8'):
            try:
                with open(file_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                logger.info(f"文件编码转换成功: {file_path} ({successful_encoding} -> UTF-8)")
            except Exception as e:
                logger.error(f"写入UTF-8文件失败 {file_path}: {e}")
                return False
        
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