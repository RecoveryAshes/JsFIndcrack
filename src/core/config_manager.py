"""
配置文件管理模块
用于管理用户自定义配置，包括请求头等
"""
import json
import os
from pathlib import Path
from typing import Dict, Any, Optional
import copy

class ConfigManager:
    """配置文件管理器"""

    DEFAULT_CONFIG = {
        "version": "1.0.0",
        "description": "JsFIndcrack 配置文件",

        # 请求头配置
        "headers": {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
            "Accept-Encoding": "gzip, deflate, br",
            "DNT": "1",
            "Connection": "keep-alive",
            "Upgrade-Insecure-Requests": "1"
        },

        # 爬虫配置
        "crawler": {
            "timeout": 30,
            "max_retries": 3,
            "delay_between_requests": 1,
            "verify_ssl": False,
            "max_file_size_mb": 50
        },

        # 浏览器配置
        "browser": {
            "engine": "playwright",  # "selenium" 或 "playwright"
            "headless": True,
            "page_load_timeout": 60,
            "window_size": "1920,1080",
            "playwright_browser": "chromium"  # "chromium", "firefox", "webkit"
        },

        # 代理配置（可选）
        "proxy": {
            "enabled": False,
            "http": "",
            "https": "",
            "no_proxy": "localhost,127.0.0.1"
        },

        # Cookie配置（可选）
        "cookies": [],

        # 相似度检测配置
        "similarity": {
            "enabled": True,
            "threshold": 0.8,
            "min_file_size": 1024
        },

        # 输出配置
        "output": {
            "base_dir": "output",
            "save_source_maps": True,
            "create_report": True,
            "log_level": "INFO"
        }
    }

    def __init__(self, config_dir: Optional[str] = None):
        """
        初始化配置管理器

        Args:
            config_dir: 配置文件目录，默认为当前目录下的config文件夹
        """
        if config_dir:
            self.config_dir = Path(config_dir)
        else:
            # 获取项目根目录
            from .config import get_base_dir
            self.config_dir = get_base_dir() / "config"

        self.config_file = self.config_dir / "config.json"
        self.custom_headers_file = self.config_dir / "custom_headers.json"
        self.config = {}

        # 初始化配置
        self._initialize_config()

    def _initialize_config(self):
        """初始化配置文件"""
        # 创建配置目录
        if not self.config_dir.exists():
            self.config_dir.mkdir(parents=True, exist_ok=True)
            print(f"✅ 创建配置目录: {self.config_dir}")

        # 创建默认配置文件
        if not self.config_file.exists():
            self._create_default_config()
            print(f"✅ 创建默认配置文件: {self.config_file}")

        # 创建自定义headers示例文件
        if not self.custom_headers_file.exists():
            self._create_custom_headers_example()
            print(f"✅ 创建自定义headers示例文件: {self.custom_headers_file}")

        # 加载配置
        self.load_config()

    def _create_default_config(self):
        """创建默认配置文件"""
        with open(self.config_file, 'w', encoding='utf-8') as f:
            json.dump(self.DEFAULT_CONFIG, f, indent=4, ensure_ascii=False)

    def _create_custom_headers_example(self):
        """创建自定义headers示例文件"""
        example_headers = {
            "description": "自定义请求头示例文件",
            "headers_sets": {
                "default": {
                    "description": "默认请求头集合",
                    "headers": self.DEFAULT_CONFIG["headers"]
                },
                "mobile": {
                    "description": "移动端请求头",
                    "headers": {
                        "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
                        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
                        "Accept-Language": "zh-CN,zh;q=0.9",
                        "Accept-Encoding": "gzip, deflate, br"
                    }
                },
                "api": {
                    "description": "API请求头",
                    "headers": {
                        "User-Agent": "JsFIndcrack/1.0",
                        "Accept": "application/json",
                        "Content-Type": "application/json"
                    }
                }
            },
            "active_set": "default"
        }

        with open(self.custom_headers_file, 'w', encoding='utf-8') as f:
            json.dump(example_headers, f, indent=4, ensure_ascii=False)

    def load_config(self) -> Dict[str, Any]:
        """加载配置文件"""
        try:
            with open(self.config_file, 'r', encoding='utf-8') as f:
                self.config = json.load(f)

            # 合并默认配置（确保新增的配置项也能被加载）
            self.config = self._merge_configs(self.DEFAULT_CONFIG, self.config)

            # 加载自定义headers
            if self.custom_headers_file.exists():
                self._load_custom_headers()

            return self.config
        except Exception as e:
            print(f"⚠️  加载配置文件失败: {e}")
            self.config = copy.deepcopy(self.DEFAULT_CONFIG)
            return self.config

    def _load_custom_headers(self):
        """加载自定义headers"""
        try:
            with open(self.custom_headers_file, 'r', encoding='utf-8') as f:
                custom_headers = json.load(f)

            # 获取活动的headers集合
            active_set = custom_headers.get('active_set', 'default')
            if active_set in custom_headers.get('headers_sets', {}):
                headers = custom_headers['headers_sets'][active_set].get('headers', {})
                if headers:
                    self.config['headers'] = headers
                    print(f"✅ 使用自定义headers集合: {active_set}")
        except Exception as e:
            print(f"⚠️  加载自定义headers失败: {e}")

    def _merge_configs(self, default: Dict, user: Dict) -> Dict:
        """
        合并默认配置和用户配置

        Args:
            default: 默认配置
            user: 用户配置

        Returns:
            合并后的配置
        """
        result = copy.deepcopy(default)

        for key, value in user.items():
            if key in result:
                if isinstance(value, dict) and isinstance(result[key], dict):
                    result[key] = self._merge_configs(result[key], value)
                else:
                    result[key] = value
            else:
                result[key] = value

        return result

    def save_config(self):
        """保存配置到文件"""
        try:
            with open(self.config_file, 'w', encoding='utf-8') as f:
                json.dump(self.config, f, indent=4, ensure_ascii=False)
            return True
        except Exception as e:
            print(f"❌ 保存配置文件失败: {e}")
            return False

    def get(self, key: str, default: Any = None) -> Any:
        """
        获取配置项

        Args:
            key: 配置项路径，支持点号分隔，如 'headers.User-Agent'
            default: 默认值

        Returns:
            配置值
        """
        keys = key.split('.')
        value = self.config

        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default

        return value

    def set(self, key: str, value: Any):
        """
        设置配置项

        Args:
            key: 配置项路径，支持点号分隔
            value: 配置值
        """
        keys = key.split('.')
        config = self.config

        for k in keys[:-1]:
            if k not in config:
                config[k] = {}
            config = config[k]

        config[keys[-1]] = value

    def get_headers(self) -> Dict[str, str]:
        """获取请求头配置"""
        return self.config.get('headers', self.DEFAULT_CONFIG['headers'])

    def get_crawler_config(self) -> Dict[str, Any]:
        """获取爬虫配置"""
        return self.config.get('crawler', self.DEFAULT_CONFIG['crawler'])

    def get_browser_config(self) -> Dict[str, Any]:
        """获取浏览器配置"""
        return self.config.get('browser', self.DEFAULT_CONFIG['browser'])

    def get_proxy_config(self) -> Dict[str, Any]:
        """获取代理配置"""
        return self.config.get('proxy', self.DEFAULT_CONFIG['proxy'])

    def get_cookies(self) -> list:
        """获取Cookie配置"""
        return self.config.get('cookies', [])

    def add_header(self, name: str, value: str):
        """
        添加或更新请求头

        Args:
            name: 请求头名称
            value: 请求头值
        """
        if 'headers' not in self.config:
            self.config['headers'] = {}
        self.config['headers'][name] = value

    def remove_header(self, name: str):
        """
        移除请求头

        Args:
            name: 请求头名称
        """
        if 'headers' in self.config and name in self.config['headers']:
            del self.config['headers'][name]

    def update_headers(self, headers: Dict[str, str]):
        """
        批量更新请求头

        Args:
            headers: 请求头字典
        """
        if 'headers' not in self.config:
            self.config['headers'] = {}
        self.config['headers'].update(headers)

    def reset_to_default(self):
        """重置为默认配置"""
        self.config = copy.deepcopy(self.DEFAULT_CONFIG)
        self.save_config()

    def validate_config(self) -> bool:
        """验证配置文件是否有效"""
        required_keys = ['headers', 'crawler', 'browser']
        for key in required_keys:
            if key not in self.config:
                return False
        return True

    def print_config(self):
        """打印当前配置"""
        print("当前配置:")
        print(json.dumps(self.config, indent=2, ensure_ascii=False))


# 全局配置管理器实例
_config_manager = None

def get_config_manager() -> ConfigManager:
    """获取全局配置管理器实例"""
    global _config_manager
    if _config_manager is None:
        _config_manager = ConfigManager()
    return _config_manager

def init_config_manager(config_dir: Optional[str] = None) -> ConfigManager:
    """
    初始化配置管理器

    Args:
        config_dir: 配置文件目录

    Returns:
        配置管理器实例
    """
    global _config_manager
    _config_manager = ConfigManager(config_dir)
    return _config_manager