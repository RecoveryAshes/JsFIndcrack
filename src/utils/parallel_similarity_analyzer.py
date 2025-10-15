#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
并行智能相似度分析器
用于高效检测反编译后JavaScript文件的相似度
"""

import os
import re
import json
import hashlib
import sys
from typing import Dict, List, Tuple, Set, Optional
from collections import defaultdict, Counter
import difflib
from dataclasses import dataclass
import multiprocessing as mp
from concurrent.futures import ProcessPoolExecutor, ThreadPoolExecutor, as_completed
import time
from functools import partial

# PyInstaller多进程保护
def is_frozen():
    """检查是否在PyInstaller打包环境中运行"""
    return getattr(sys, 'frozen', False) and hasattr(sys, '_MEIPASS')

def get_safe_executor(max_workers=None):
    """获取安全的执行器，在PyInstaller环境中使用ThreadPoolExecutor"""
    if is_frozen():
        # 在打包环境中使用线程池避免多进程问题
        # 限制线程数以避免资源竞争
        safe_max_workers = min(max_workers or 4, 8)
        return ThreadPoolExecutor(max_workers=safe_max_workers)
    else:
        # 在开发环境中使用进程池获得更好的性能
        return ProcessPoolExecutor(max_workers=max_workers)


@dataclass
class SimilarityResult:
    """相似度检测结果"""
    file1: str
    file2: str
    overall_similarity: float
    structural_similarity: float
    content_similarity: float
    function_similarity: float
    variable_similarity: float
    is_duplicate: bool
    similarity_reasons: List[str]


class FastJavaScriptAnalyzer:
    """快速JavaScript代码分析器"""
    
    def __init__(self):
        # 预编译正则表达式以提高性能
        self.function_pattern = re.compile(r'function\s+(\w+)\s*\([^)]*\)\s*{', re.MULTILINE)
        self.variable_pattern = re.compile(r'(?:var|let|const)\s+(\w+)', re.MULTILINE)
        self.import_pattern = re.compile(r'(?:import|require)\s*\(["\']([^"\']+)["\']', re.MULTILINE)
        self.webpack_module_pattern = re.compile(r'"([a-f0-9]+)":\s*function\s*\([^)]*\)', re.MULTILINE)
        self.comment_pattern = re.compile(r'//.*$|/\*.*?\*/', re.MULTILINE | re.DOTALL)
        self.whitespace_pattern = re.compile(r'\s+')
        
        # 结构特征模式
        self.structure_patterns = {
            'if_statements': re.compile(r'\bif\s*\('),
            'for_loops': re.compile(r'\bfor\s*\('),
            'while_loops': re.compile(r'\bwhile\s*\('),
            'try_catch': re.compile(r'\btry\s*{'),
            'object_literals': re.compile(r'{[^}]*}'),
            'array_literals': re.compile(r'\[[^\]]*\]'),
            'arrow_functions': re.compile(r'=>'),
            'async_await': re.compile(r'\b(async|await)\b'),
        }
    
    def extract_features(self, content: str) -> Dict:
        """一次性提取所有特征以提高效率"""
        features = {
            'functions': set(),
            'variables': set(),
            'imports': set(),
            'webpack_modules': set(),
            'structure_counts': {},
            'content_hash': '',
            'normalized_content': '',
            'file_size': len(content),
            'line_count': content.count('\n') + 1
        }
        
        # 提取函数名
        features['functions'] = set(self.function_pattern.findall(content))
        
        # 提取变量名
        features['variables'] = set(self.variable_pattern.findall(content))
        
        # 提取导入模块
        features['imports'] = set(self.import_pattern.findall(content))
        
        # 提取webpack模块
        features['webpack_modules'] = set(self.webpack_module_pattern.findall(content))
        
        # 计算结构特征
        for name, pattern in self.structure_patterns.items():
            features['structure_counts'][name] = len(pattern.findall(content))
        
        # 标准化内容
        normalized = self.comment_pattern.sub('', content)
        normalized = self.whitespace_pattern.sub(' ', normalized).strip()
        features['normalized_content'] = normalized
        
        # 计算内容哈希
        features['content_hash'] = hashlib.md5(normalized.encode('utf-8')).hexdigest()
        
        return features


def load_file_content(file_path: str) -> Tuple[str, str]:
    """加载文件内容"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        return file_path, content
    except Exception:
        try:
            with open(file_path, 'r', encoding='latin-1') as f:
                content = f.read()
            return file_path, content
        except Exception as e:
            print(f"无法读取文件 {file_path}: {e}")
            return file_path, ""


def extract_file_features(file_path: str) -> Tuple[str, Dict]:
    """提取单个文件的特征"""
    _, content = load_file_content(file_path)
    if not content:
        return file_path, {}
    
    analyzer = FastJavaScriptAnalyzer()
    features = analyzer.extract_features(content)
    return file_path, features


def calculate_similarity_pair(args: Tuple[str, str, Dict, Dict, float]) -> Optional[SimilarityResult]:
    """计算两个文件的相似度"""
    file1, file2, features1, features2, threshold = args
    
    if not features1 or not features2:
        return None
    
    # 快速哈希检查
    if features1['content_hash'] == features2['content_hash']:
        return SimilarityResult(
            file1=file1,
            file2=file2,
            overall_similarity=1.0,
            structural_similarity=1.0,
            content_similarity=1.0,
            function_similarity=1.0,
            variable_similarity=1.0,
            is_duplicate=True,
            similarity_reasons=["完全相同的内容哈希"]
        )
    
    # 快速文件大小检查
    size_ratio = min(features1['file_size'], features2['file_size']) / max(features1['file_size'], features2['file_size'])
    if size_ratio < 0.5:  # 文件大小差异太大，跳过详细比较
        return None
    
    # 计算各种相似度
    content_sim = calculate_content_similarity(features1, features2)
    structural_sim = calculate_structural_similarity(features1, features2)
    function_sim = calculate_set_similarity(features1['functions'], features2['functions'])
    variable_sim = calculate_set_similarity(features1['variables'], features2['variables'])
    
    # 计算综合相似度（加权平均）
    weights = {'content': 0.4, 'structural': 0.3, 'function': 0.15, 'variable': 0.15}
    overall_sim = (
        content_sim * weights['content'] +
        structural_sim * weights['structural'] +
        function_sim * weights['function'] +
        variable_sim * weights['variable']
    )
    
    # 只返回相似度较高的结果
    if overall_sim < threshold * 0.7:  # 预筛选阈值
        return None
    
    # 判断是否为重复文件
    is_duplicate = overall_sim >= threshold
    
    # 生成相似度原因
    reasons = []
    if content_sim > 0.9:
        reasons.append(f"内容高度相似 ({content_sim:.2f})")
    if structural_sim > 0.9:
        reasons.append(f"结构高度相似 ({structural_sim:.2f})")
    if function_sim > 0.8:
        reasons.append(f"函数名高度重叠 ({function_sim:.2f})")
    if variable_sim > 0.8:
        reasons.append(f"变量名高度重叠 ({variable_sim:.2f})")
    
    return SimilarityResult(
        file1=file1,
        file2=file2,
        overall_similarity=overall_sim,
        structural_similarity=structural_sim,
        content_similarity=content_sim,
        function_similarity=function_sim,
        variable_similarity=variable_sim,
        is_duplicate=is_duplicate,
        similarity_reasons=reasons
    )


def calculate_content_similarity(features1: Dict, features2: Dict) -> float:
    """计算内容相似度"""
    content1 = features1['normalized_content']
    content2 = features2['normalized_content']
    
    if not content1 or not content2:
        return 0.0
    
    # 使用快速相似度计算
    if len(content1) > 10000 or len(content2) > 10000:
        # 对于大文件，使用采样比较
        sample_size = min(1000, len(content1), len(content2))
        content1 = content1[:sample_size]
        content2 = content2[:sample_size]
    
    return difflib.SequenceMatcher(None, content1, content2).ratio()


def calculate_structural_similarity(features1: Dict, features2: Dict) -> float:
    """计算结构相似度"""
    struct1 = features1['structure_counts']
    struct2 = features2['structure_counts']
    
    if not struct1 and not struct2:
        return 1.0
    if not struct1 or not struct2:
        return 0.0
    
    # 计算结构特征的余弦相似度
    all_keys = set(struct1.keys()) | set(struct2.keys())
    
    dot_product = sum(struct1.get(key, 0) * struct2.get(key, 0) for key in all_keys)
    norm1 = sum(struct1.get(key, 0) ** 2 for key in all_keys) ** 0.5
    norm2 = sum(struct2.get(key, 0) ** 2 for key in all_keys) ** 0.5
    
    if norm1 == 0 or norm2 == 0:
        return 0.0
    
    return dot_product / (norm1 * norm2)


def calculate_set_similarity(set1: Set, set2: Set) -> float:
    """计算集合相似度"""
    if not set1 and not set2:
        return 1.0
    if not set1 or not set2:
        return 0.0
    
    intersection = len(set1 & set2)
    union = len(set1 | set2)
    
    return intersection / union if union > 0 else 0.0


class ParallelDeduplicationProcessor:
    """并行去重处理器"""
    
    def __init__(self, similarity_threshold: float = 0.8, max_workers: Optional[int] = None):
        self.similarity_threshold = similarity_threshold
        self.max_workers = max_workers or min(mp.cpu_count(), 8)  # 限制最大进程数
        
    def process_directory(self, input_dir: str, output_dir: str) -> Dict:
        """并行处理目录中的所有文件进行去重"""
        start_time = time.time()
        
        if not os.path.exists(output_dir):
            os.makedirs(output_dir)
        
        # 获取所有JavaScript文件
        js_files = []
        for root, dirs, files in os.walk(input_dir):
            for file in files:
                if file.endswith('.js'):
                    js_files.append(os.path.join(root, file))
        
        print(f"找到 {len(js_files)} 个JavaScript文件")
        
        # 第一阶段：并行提取所有文件的特征
        print("第一阶段：提取文件特征...")
        features_dict = {}
        
        with get_safe_executor(max_workers=self.max_workers) as executor:
            future_to_file = {executor.submit(extract_file_features, file_path): file_path 
                             for file_path in js_files}
            
            completed = 0
            for future in as_completed(future_to_file):
                file_path, features = future.result()
                if features:
                    features_dict[file_path] = features
                completed += 1
                if completed % 10 == 0:
                    print(f"已处理 {completed}/{len(js_files)} 个文件的特征提取")
        
        print(f"特征提取完成，有效文件数: {len(features_dict)}")
        
        # 第二阶段：快速哈希去重
        print("第二阶段：快速哈希去重...")
        hash_groups = defaultdict(list)
        for file_path, features in features_dict.items():
            content_hash = features['content_hash']
            hash_groups[content_hash].append(file_path)
        
        # 找出哈希相同的文件组
        exact_duplicate_groups = [group for group in hash_groups.values() if len(group) > 1]
        unique_files = [group[0] for group in hash_groups.values()]  # 每组取一个代表
        
        print(f"发现 {len(exact_duplicate_groups)} 个完全相同的文件组")
        print(f"去重后剩余 {len(unique_files)} 个唯一文件需要进一步比较")
        
        # 第三阶段：并行相似度比较（只比较唯一文件）
        print("第三阶段：相似度分析...")
        similarity_results = []
        
        # 生成文件对
        file_pairs = []
        for i, file1 in enumerate(unique_files):
            for j, file2 in enumerate(unique_files[i+1:], i+1):
                file_pairs.append((file1, file2, features_dict[file1], features_dict[file2], self.similarity_threshold))
        
        print(f"需要比较 {len(file_pairs)} 个文件对")
        
        # 并行计算相似度
        with get_safe_executor(max_workers=self.max_workers) as executor:
            future_to_pair = {executor.submit(calculate_similarity_pair, pair): pair 
                             for pair in file_pairs}
            
            completed = 0
            for future in as_completed(future_to_pair):
                result = future.result()
                if result:
                    similarity_results.append(result)
                completed += 1
                if completed % 100 == 0:
                    print(f"已比较 {completed}/{len(file_pairs)} 个文件对")
        
        # 第四阶段：构建相似文件组
        print("第四阶段：构建相似文件组...")
        similar_groups = self._build_similarity_groups(similarity_results, unique_files)
        
        # 第五阶段：生成输出
        print("第五阶段：生成输出...")
        self._generate_output(input_dir, output_dir, exact_duplicate_groups, similar_groups, 
                            hash_groups, similarity_results, js_files)
        
        end_time = time.time()
        processing_time = end_time - start_time
        
        # 生成最终报告
        total_duplicates = sum(len(group) - 1 for group in exact_duplicate_groups)
        total_similar = sum(len(group) - 1 for group in similar_groups)
        
        report = {
            'total_files': len(js_files),
            'exact_duplicate_groups': len(exact_duplicate_groups),
            'similar_groups': len(similar_groups),
            'total_exact_duplicates': total_duplicates,
            'total_similar_files': total_similar,
            'unique_files': len(js_files) - total_duplicates - total_similar,
            'processing_time_seconds': processing_time,
            'files_per_second': len(js_files) / processing_time if processing_time > 0 else 0
        }
        
        print(f"\n处理完成! 耗时: {processing_time:.2f} 秒")
        print(f"处理速度: {report['files_per_second']:.1f} 文件/秒")
        print(f"- 总文件数: {report['total_files']}")
        print(f"- 完全相同文件组: {report['exact_duplicate_groups']}")
        print(f"- 相似文件组: {report['similar_groups']}")
        print(f"- 唯一文件数: {report['unique_files']}")
        
        return report
    
    def _build_similarity_groups(self, similarity_results: List[SimilarityResult], unique_files: List[str]) -> List[List[str]]:
        """构建相似文件组"""
        # 使用并查集算法构建相似文件组
        parent = {file: file for file in unique_files}
        
        def find(x):
            if parent[x] != x:
                parent[x] = find(parent[x])
            return parent[x]
        
        def union(x, y):
            px, py = find(x), find(y)
            if px != py:
                parent[px] = py
        
        # 合并相似的文件
        for result in similarity_results:
            if result.is_duplicate:
                union(result.file1, result.file2)
        
        # 构建组
        groups = defaultdict(list)
        for file in unique_files:
            root = find(file)
            groups[root].append(file)
        
        # 只返回包含多个文件的组
        return [group for group in groups.values() if len(group) > 1]
    
    def _generate_output(self, input_dir: str, output_dir: str, exact_groups: List[List[str]], 
                        similar_groups: List[List[str]], hash_groups: Dict, 
                        similarity_results: List[SimilarityResult], all_files: List[str]):
        """生成输出文件和目录"""
        # 创建输出目录结构
        unique_dir = os.path.join(output_dir, 'unique')
        exact_duplicates_dir = os.path.join(output_dir, 'exact_duplicates')
        similar_dir = os.path.join(output_dir, 'similar_groups')
        merged_dir = os.path.join(output_dir, 'merged')  # 新增：合并目录
        
        os.makedirs(unique_dir, exist_ok=True)
        os.makedirs(exact_duplicates_dir, exist_ok=True)
        os.makedirs(similar_dir, exist_ok=True)
        os.makedirs(merged_dir, exist_ok=True)  # 新增：创建合并目录
        
        # 收集所有重复和相似的文件
        all_duplicates = set()
        for group in exact_groups:
            all_duplicates.update(group)
        for group in similar_groups:
            all_duplicates.update(group)
        
        # 复制唯一文件
        unique_files = [f for f in all_files if f not in all_duplicates]
        for file_path in unique_files:
            filename = os.path.basename(file_path)
            dest_path = os.path.join(unique_dir, filename)
            self._copy_file(file_path, dest_path)
            # 新增：同时复制到合并目录
            merged_dest_path = os.path.join(merged_dir, filename)
            self._copy_file(file_path, merged_dest_path)
        
        # 处理完全相同的文件组
        for i, group in enumerate(exact_groups):
            group_dir = os.path.join(exact_duplicates_dir, f'exact_group_{i+1}')
            os.makedirs(group_dir, exist_ok=True)
            for file_path in group:
                filename = os.path.basename(file_path)
                dest_path = os.path.join(group_dir, filename)
                self._copy_file(file_path, dest_path)
            
            # 新增：将每组的第一个文件（代表文件）复制到合并目录
            if group:
                representative_file = group[0]
                filename = os.path.basename(representative_file)
                merged_dest_path = os.path.join(merged_dir, filename)
                self._copy_file(representative_file, merged_dest_path)
        
        # 处理相似文件组
        for i, group in enumerate(similar_groups):
            group_dir = os.path.join(similar_dir, f'similar_group_{i+1}')
            os.makedirs(group_dir, exist_ok=True)
            for file_path in group:
                filename = os.path.basename(file_path)
                dest_path = os.path.join(group_dir, filename)
                self._copy_file(file_path, dest_path)
            
            # 新增：将每组的第一个文件（代表文件）复制到合并目录
            if group:
                representative_file = group[0]
                filename = os.path.basename(representative_file)
                merged_dest_path = os.path.join(merged_dir, filename)
                self._copy_file(representative_file, merged_dest_path)
        
        # 生成详细报告
        report = {
            'exact_duplicate_groups': [
                [os.path.basename(f) for f in group] for group in exact_groups
            ],
            'similar_groups': [
                [os.path.basename(f) for f in group] for group in similar_groups
            ],
            'similarity_details': [
                {
                    'file1': os.path.basename(r.file1),
                    'file2': os.path.basename(r.file2),
                    'overall_similarity': r.overall_similarity,
                    'is_duplicate': r.is_duplicate,
                    'reasons': r.similarity_reasons
                }
                for r in similarity_results if r.is_duplicate
            ]
        }
        
        report_path = os.path.join(output_dir, 'detailed_similarity_report.json')
        with open(report_path, 'w', encoding='utf-8') as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
    
    def _copy_file(self, src: str, dest: str):
        """复制文件"""
        try:
            with open(src, 'r', encoding='utf-8') as f:
                content = f.read()
            with open(dest, 'w', encoding='utf-8') as f:
                f.write(content)
        except Exception:
            try:
                with open(src, 'r', encoding='latin-1') as f:
                    content = f.read()
                with open(dest, 'w', encoding='utf-8') as f:
                    f.write(content)
            except Exception as e:
                print(f"复制文件失败 {src} -> {dest}: {e}")


if __name__ == "__main__":
    # 多进程保护 - 在PyInstaller环境中完全避免设置start_method
    if not is_frozen():
        try:
            # 只在开发环境中设置start_method
            mp.set_start_method('spawn', force=True)
        except RuntimeError:
            # 如果已经设置过start_method，忽略错误
            pass
    # 在打包环境中不设置start_method，避免子进程问题
    
    # 示例用法
    processor = ParallelDeduplicationProcessor(similarity_threshold=0.8, max_workers=8)
    
    input_directory = "/Users/recovery/opt/tools/自编写小工具/JsFIndcrack/output/218.108.73.58_3000/decode"
    output_directory = "/Users/recovery/opt/tools/自编写小工具/JsFIndcrack/output/218.108.73.58_3000/parallel_similarity_analysis"
    
    processor.process_directory(input_directory, output_directory)