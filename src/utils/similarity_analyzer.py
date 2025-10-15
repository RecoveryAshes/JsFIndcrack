#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
智能相似度分析器
用于检测反编译后JavaScript文件的相似度
"""

import os
import re
import json
import hashlib
from typing import Dict, List, Tuple, Set, Optional
from collections import defaultdict, Counter
import difflib
from dataclasses import dataclass
import ast
import esprima  # JavaScript AST解析器


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


class JavaScriptAnalyzer:
    """JavaScript代码分析器"""
    
    def __init__(self):
        self.function_pattern = re.compile(r'function\s+(\w+)\s*\([^)]*\)\s*{', re.MULTILINE)
        self.variable_pattern = re.compile(r'(?:var|let|const)\s+(\w+)', re.MULTILINE)
        self.import_pattern = re.compile(r'(?:import|require)\s*\(["\']([^"\']+)["\']', re.MULTILINE)
        self.webpack_module_pattern = re.compile(r'"([a-f0-9]+)":\s*function\s*\([^)]*\)', re.MULTILINE)
        
    def extract_functions(self, content: str) -> Set[str]:
        """提取函数名"""
        functions = set()
        matches = self.function_pattern.findall(content)
        functions.update(matches)
        return functions
    
    def extract_variables(self, content: str) -> Set[str]:
        """提取变量名"""
        variables = set()
        matches = self.variable_pattern.findall(content)
        variables.update(matches)
        return variables
    
    def extract_imports(self, content: str) -> Set[str]:
        """提取导入模块"""
        imports = set()
        matches = self.import_pattern.findall(content)
        imports.update(matches)
        return imports
    
    def extract_webpack_modules(self, content: str) -> Set[str]:
        """提取webpack模块ID"""
        modules = set()
        matches = self.webpack_module_pattern.findall(content)
        modules.update(matches)
        return modules
    
    def get_ast_structure(self, content: str) -> Optional[Dict]:
        """获取AST结构特征"""
        try:
            # 尝试解析JavaScript AST
            ast_tree = esprima.parseScript(content, {'tolerant': True})
            return self._extract_ast_features(ast_tree)
        except Exception:
            # 如果解析失败，使用正则表达式提取基本结构
            return self._extract_basic_structure(content)
    
    def _extract_ast_features(self, ast_tree) -> Dict:
        """从AST中提取特征"""
        features = {
            'node_types': Counter(),
            'function_count': 0,
            'variable_count': 0,
            'depth': 0
        }
        
        def traverse(node, depth=0):
            if hasattr(node, 'type'):
                features['node_types'][node.type] += 1
                features['depth'] = max(features['depth'], depth)
                
                if node.type == 'FunctionDeclaration':
                    features['function_count'] += 1
                elif node.type == 'VariableDeclaration':
                    features['variable_count'] += 1
            
            # 递归遍历子节点
            if hasattr(node, '__dict__'):
                for key, value in node.__dict__.items():
                    if isinstance(value, list):
                        for item in value:
                            if hasattr(item, 'type'):
                                traverse(item, depth + 1)
                    elif hasattr(value, 'type'):
                        traverse(value, depth + 1)
        
        traverse(ast_tree)
        return features
    
    def _extract_basic_structure(self, content: str) -> Dict:
        """使用正则表达式提取基本结构特征"""
        features = {
            'node_types': Counter(),
            'function_count': len(self.function_pattern.findall(content)),
            'variable_count': len(self.variable_pattern.findall(content)),
            'depth': content.count('{')  # 简单的嵌套深度估算
        }
        
        # 统计常见的JavaScript结构
        patterns = {
            'if_statements': r'\bif\s*\(',
            'for_loops': r'\bfor\s*\(',
            'while_loops': r'\bwhile\s*\(',
            'try_catch': r'\btry\s*{',
            'object_literals': r'{[^}]*}',
            'array_literals': r'\[[^\]]*\]'
        }
        
        for pattern_name, pattern in patterns.items():
            features['node_types'][pattern_name] = len(re.findall(pattern, content))
        
        return features


class SimilarityAnalyzer:
    """智能相似度分析器"""
    
    def __init__(self, similarity_threshold: float = 0.8):
        self.similarity_threshold = similarity_threshold
        self.js_analyzer = JavaScriptAnalyzer()
        
    def calculate_content_similarity(self, content1: str, content2: str) -> float:
        """计算内容相似度（基于文本差异）"""
        # 移除空白字符和注释进行比较
        normalized1 = self._normalize_content(content1)
        normalized2 = self._normalize_content(content2)
        
        if not normalized1 or not normalized2:
            return 0.0
        
        # 使用difflib计算相似度
        similarity = difflib.SequenceMatcher(None, normalized1, normalized2).ratio()
        return similarity
    
    def calculate_structural_similarity(self, content1: str, content2: str) -> float:
        """计算结构相似度（基于AST）"""
        ast1 = self.js_analyzer.get_ast_structure(content1)
        ast2 = self.js_analyzer.get_ast_structure(content2)
        
        if not ast1 or not ast2:
            return 0.0
        
        # 比较AST特征
        similarities = []
        
        # 比较节点类型分布
        if ast1.get('node_types') and ast2.get('node_types'):
            node_sim = self._calculate_counter_similarity(
                ast1['node_types'], ast2['node_types']
            )
            similarities.append(node_sim)
        
        # 比较函数和变量数量
        func_sim = self._calculate_numeric_similarity(
            ast1.get('function_count', 0), ast2.get('function_count', 0)
        )
        var_sim = self._calculate_numeric_similarity(
            ast1.get('variable_count', 0), ast2.get('variable_count', 0)
        )
        depth_sim = self._calculate_numeric_similarity(
            ast1.get('depth', 0), ast2.get('depth', 0)
        )
        
        similarities.extend([func_sim, var_sim, depth_sim])
        
        return sum(similarities) / len(similarities) if similarities else 0.0
    
    def calculate_function_similarity(self, content1: str, content2: str) -> float:
        """计算函数相似度"""
        functions1 = self.js_analyzer.extract_functions(content1)
        functions2 = self.js_analyzer.extract_functions(content2)
        
        if not functions1 and not functions2:
            return 1.0
        if not functions1 or not functions2:
            return 0.0
        
        # 计算函数名的交集比例
        intersection = len(functions1 & functions2)
        union = len(functions1 | functions2)
        
        return intersection / union if union > 0 else 0.0
    
    def calculate_variable_similarity(self, content1: str, content2: str) -> float:
        """计算变量相似度"""
        variables1 = self.js_analyzer.extract_variables(content1)
        variables2 = self.js_analyzer.extract_variables(content2)
        
        if not variables1 and not variables2:
            return 1.0
        if not variables1 or not variables2:
            return 0.0
        
        # 计算变量名的交集比例
        intersection = len(variables1 & variables2)
        union = len(variables1 | variables2)
        
        return intersection / union if union > 0 else 0.0
    
    def analyze_similarity(self, file1_path: str, file2_path: str) -> SimilarityResult:
        """分析两个文件的相似度"""
        try:
            with open(file1_path, 'r', encoding='utf-8') as f:
                content1 = f.read()
        except Exception:
            with open(file1_path, 'r', encoding='latin-1') as f:
                content1 = f.read()
        
        try:
            with open(file2_path, 'r', encoding='utf-8') as f:
                content2 = f.read()
        except Exception:
            with open(file2_path, 'r', encoding='latin-1') as f:
                content2 = f.read()
        
        # 计算各种相似度
        content_sim = self.calculate_content_similarity(content1, content2)
        structural_sim = self.calculate_structural_similarity(content1, content2)
        function_sim = self.calculate_function_similarity(content1, content2)
        variable_sim = self.calculate_variable_similarity(content1, content2)
        
        # 计算综合相似度（加权平均）
        weights = {
            'content': 0.3,
            'structural': 0.3,
            'function': 0.2,
            'variable': 0.2
        }
        
        overall_sim = (
            content_sim * weights['content'] +
            structural_sim * weights['structural'] +
            function_sim * weights['function'] +
            variable_sim * weights['variable']
        )
        
        # 判断是否为重复文件
        is_duplicate = overall_sim >= self.similarity_threshold
        
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
            file1=file1_path,
            file2=file2_path,
            overall_similarity=overall_sim,
            structural_similarity=structural_sim,
            content_similarity=content_sim,
            function_similarity=function_sim,
            variable_similarity=variable_sim,
            is_duplicate=is_duplicate,
            similarity_reasons=reasons
        )
    
    def _normalize_content(self, content: str) -> str:
        """标准化内容（移除空白和注释）"""
        # 移除单行注释
        content = re.sub(r'//.*$', '', content, flags=re.MULTILINE)
        # 移除多行注释
        content = re.sub(r'/\*.*?\*/', '', content, flags=re.DOTALL)
        # 移除多余的空白字符
        content = re.sub(r'\s+', ' ', content)
        return content.strip()
    
    def _calculate_counter_similarity(self, counter1: Counter, counter2: Counter) -> float:
        """计算Counter对象的相似度"""
        if not counter1 and not counter2:
            return 1.0
        if not counter1 or not counter2:
            return 0.0
        
        # 计算余弦相似度
        all_keys = set(counter1.keys()) | set(counter2.keys())
        
        dot_product = sum(counter1.get(key, 0) * counter2.get(key, 0) for key in all_keys)
        norm1 = sum(counter1.get(key, 0) ** 2 for key in all_keys) ** 0.5
        norm2 = sum(counter2.get(key, 0) ** 2 for key in all_keys) ** 0.5
        
        if norm1 == 0 or norm2 == 0:
            return 0.0
        
        return dot_product / (norm1 * norm2)
    
    def _calculate_numeric_similarity(self, num1: int, num2: int) -> float:
        """计算数值相似度"""
        if num1 == 0 and num2 == 0:
            return 1.0
        if num1 == 0 or num2 == 0:
            return 0.0
        
        return min(num1, num2) / max(num1, num2)


class DeduplicationProcessor:
    """去重处理器"""
    
    def __init__(self, similarity_threshold: float = 0.8):
        self.analyzer = SimilarityAnalyzer(similarity_threshold)
        self.similarity_threshold = similarity_threshold
    
    def process_directory(self, input_dir: str, output_dir: str) -> Dict:
        """处理目录中的所有文件进行去重"""
        if not os.path.exists(output_dir):
            os.makedirs(output_dir)
        
        # 获取所有JavaScript文件
        js_files = []
        for root, dirs, files in os.walk(input_dir):
            for file in files:
                if file.endswith('.js'):
                    js_files.append(os.path.join(root, file))
        
        print(f"找到 {len(js_files)} 个JavaScript文件")
        
        # 存储相似度结果
        similarity_results = []
        duplicate_groups = []
        processed_files = set()
        
        # 两两比较文件
        for i, file1 in enumerate(js_files):
            if file1 in processed_files:
                continue
                
            current_group = [file1]
            
            for j, file2 in enumerate(js_files[i+1:], i+1):
                if file2 in processed_files:
                    continue
                
                print(f"比较文件 {i+1}/{len(js_files)}: {os.path.basename(file1)} vs {os.path.basename(file2)}")
                
                result = self.analyzer.analyze_similarity(file1, file2)
                similarity_results.append(result)
                
                if result.is_duplicate:
                    current_group.append(file2)
                    processed_files.add(file2)
            
            if len(current_group) > 1:
                duplicate_groups.append(current_group)
            
            processed_files.add(file1)
        
        # 创建输出结构
        unique_dir = os.path.join(output_dir, 'unique')
        duplicates_dir = os.path.join(output_dir, 'duplicates')
        os.makedirs(unique_dir, exist_ok=True)
        os.makedirs(duplicates_dir, exist_ok=True)
        
        # 复制唯一文件
        unique_files = []
        duplicate_files = []
        
        all_duplicates = set()
        for group in duplicate_groups:
            all_duplicates.update(group)
            duplicate_files.extend(group)
        
        for file_path in js_files:
            if file_path not in all_duplicates:
                unique_files.append(file_path)
        
        # 复制唯一文件到unique目录
        for file_path in unique_files:
            filename = os.path.basename(file_path)
            dest_path = os.path.join(unique_dir, filename)
            self._copy_file(file_path, dest_path)
        
        # 处理重复文件组
        for i, group in enumerate(duplicate_groups):
            group_dir = os.path.join(duplicates_dir, f'group_{i+1}')
            os.makedirs(group_dir, exist_ok=True)
            
            for file_path in group:
                filename = os.path.basename(file_path)
                dest_path = os.path.join(group_dir, filename)
                self._copy_file(file_path, dest_path)
        
        # 生成报告
        report = {
            'total_files': len(js_files),
            'unique_files': len(unique_files),
            'duplicate_groups': len(duplicate_groups),
            'total_duplicates': len(duplicate_files),
            'similarity_results': [
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
        
        # 保存报告
        report_path = os.path.join(output_dir, 'similarity_report.json')
        with open(report_path, 'w', encoding='utf-8') as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
        
        print(f"\n处理完成:")
        print(f"- 总文件数: {report['total_files']}")
        print(f"- 唯一文件数: {report['unique_files']}")
        print(f"- 重复文件组数: {report['duplicate_groups']}")
        print(f"- 重复文件总数: {report['total_duplicates']}")
        print(f"- 报告已保存到: {report_path}")
        
        return report
    
    def _copy_file(self, src: str, dest: str):
        """复制文件"""
        try:
            with open(src, 'r', encoding='utf-8') as f:
                content = f.read()
            with open(dest, 'w', encoding='utf-8') as f:
                f.write(content)
        except Exception:
            # 如果UTF-8失败，尝试其他编码
            try:
                with open(src, 'r', encoding='latin-1') as f:
                    content = f.read()
                with open(dest, 'w', encoding='utf-8') as f:
                    f.write(content)
            except Exception as e:
                print(f"复制文件失败 {src} -> {dest}: {e}")


if __name__ == "__main__":
    # 示例用法
    processor = DeduplicationProcessor(similarity_threshold=0.8)
    
    input_directory = "/Users/recovery/opt/tools/自编写小工具/JsFIndcrack/output/218.108.73.58_3000/decode"
    output_directory = "/Users/recovery/opt/tools/自编写小工具/JsFIndcrack/output/218.108.73.58_3000/similarity_analysis"
    
    processor.process_directory(input_directory, output_directory)