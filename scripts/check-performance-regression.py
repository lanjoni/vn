#!/usr/bin/env python3
"""
Performance regression detection script for test suite.
Compares current performance metrics with baseline and fails if regression is detected.
"""

import sys
import re
import json
import argparse
from typing import Dict, Optional, Tuple


def parse_performance_output(output: str) -> Dict[str, float]:
    """Parse performance measurement output and extract timing data."""
    metrics = {}
    
    # Extract timing information from the output
    patterns = {
        'unit_tests': r'1\. Unit Tests Performance:\s*real\s+(\d+)m([\d.]+)s',
        'shared_tests': r'2\. Shared Tests Performance:\s*real\s+(\d+)m([\d.]+)s',
        'integration_tests': r'3\. Integration Tests Performance:\s*real\s+(\d+)m([\d.]+)s',
        'e2e_tests': r'4\. E2E Tests Performance:\s*real\s+(\d+)m([\d.]+)s',
        'all_tests': r'5\. All Tests Combined:\s*real\s+(\d+)m([\d.]+)s'
    }
    
    for test_type, pattern in patterns.items():
        match = re.search(pattern, output)
        if match:
            minutes = int(match.group(1))
            seconds = float(match.group(2))
            total_seconds = minutes * 60 + seconds
            metrics[test_type] = total_seconds
    
    return metrics


def load_baseline(baseline_file: str) -> Dict[str, float]:
    """Load baseline performance metrics from file."""
    try:
        with open(baseline_file, 'r') as f:
            content = f.read()
        return parse_performance_output(content)
    except FileNotFoundError:
        print(f"Warning: Baseline file {baseline_file} not found")
        return {}
    except Exception as e:
        print(f"Error loading baseline: {e}")
        return {}


def load_current(current_file: str) -> Dict[str, float]:
    """Load current performance metrics from file."""
    try:
        with open(current_file, 'r') as f:
            content = f.read()
        return parse_performance_output(content)
    except Exception as e:
        print(f"Error loading current metrics: {e}")
        return {}


def check_regression(baseline: Dict[str, float], current: Dict[str, float], 
                    threshold: float) -> Tuple[bool, Dict[str, Dict]]:
    """
    Check for performance regression.
    
    Args:
        baseline: Baseline performance metrics
        current: Current performance metrics  
        threshold: Regression threshold (e.g., 1.5 for 50% slower)
    
    Returns:
        Tuple of (has_regression, detailed_results)
    """
    results = {}
    has_regression = False
    
    for test_type in current.keys():
        if test_type not in baseline:
            results[test_type] = {
                'status': 'new',
                'current': current[test_type],
                'baseline': None,
                'ratio': None
            }
            continue
            
        baseline_time = baseline[test_type]
        current_time = current[test_type]
        ratio = current_time / baseline_time if baseline_time > 0 else float('inf')
        
        status = 'improved' if ratio < 1.0 else 'stable'
        if ratio > threshold:
            status = 'regression'
            has_regression = True
        
        results[test_type] = {
            'status': status,
            'current': current_time,
            'baseline': baseline_time,
            'ratio': ratio
        }
    
    return has_regression, results


def format_time(seconds: float) -> str:
    """Format seconds as human-readable time."""
    if seconds < 60:
        return f"{seconds:.1f}s"
    else:
        minutes = int(seconds // 60)
        secs = seconds % 60
        return f"{minutes}m{secs:.1f}s"


def print_results(results: Dict[str, Dict], threshold: float):
    """Print formatted regression check results."""
    print("\n=== Performance Regression Analysis ===")
    print(f"Regression threshold: {threshold:.1%} slower than baseline\n")
    
    # Group results by status
    regressions = []
    improvements = []
    stable = []
    new_tests = []
    
    for test_type, data in results.items():
        if data['status'] == 'regression':
            regressions.append((test_type, data))
        elif data['status'] == 'improved':
            improvements.append((test_type, data))
        elif data['status'] == 'stable':
            stable.append((test_type, data))
        else:
            new_tests.append((test_type, data))
    
    # Print regressions (most important)
    if regressions:
        print("🚨 PERFORMANCE REGRESSIONS DETECTED:")
        for test_type, data in regressions:
            current_str = format_time(data['current'])
            baseline_str = format_time(data['baseline'])
            ratio_str = f"{data['ratio']:.1%}" if data['ratio'] < 10 else ">10x"
            print(f"  ❌ {test_type}: {current_str} (was {baseline_str}) - {ratio_str} slower")
        print()
    
    # Print improvements
    if improvements:
        print("✅ PERFORMANCE IMPROVEMENTS:")
        for test_type, data in improvements:
            current_str = format_time(data['current'])
            baseline_str = format_time(data['baseline'])
            improvement = (1 - data['ratio']) * 100
            print(f"  🚀 {test_type}: {current_str} (was {baseline_str}) - {improvement:.1f}% faster")
        print()
    
    # Print stable tests
    if stable:
        print("📊 STABLE PERFORMANCE:")
        for test_type, data in stable:
            current_str = format_time(data['current'])
            baseline_str = format_time(data['baseline'])
            change = (data['ratio'] - 1) * 100
            change_str = f"{change:+.1f}%" if abs(change) > 0.1 else "±0%"
            print(f"  ✓ {test_type}: {current_str} (was {baseline_str}) {change_str}")
        print()
    
    # Print new tests
    if new_tests:
        print("🆕 NEW TESTS:")
        for test_type, data in new_tests:
            current_str = format_time(data['current'])
            print(f"  + {test_type}: {current_str}")
        print()


def main():
    parser = argparse.ArgumentParser(description='Check for test performance regressions')
    parser.add_argument('baseline', help='Baseline performance file')
    parser.add_argument('current', help='Current performance file')
    parser.add_argument('--threshold', type=float, default=1.5, 
                       help='Regression threshold (default: 1.5 = 50% slower)')
    parser.add_argument('--fail-on-regression', action='store_true',
                       help='Exit with error code if regression detected')
    parser.add_argument('--json-output', help='Save detailed results to JSON file')
    
    args = parser.parse_args()
    
    # Load metrics
    baseline = load_baseline(args.baseline)
    current = load_current(args.current)
    
    if not baseline:
        print("No baseline data available - treating current run as new baseline")
        print_results({k: {'status': 'new', 'current': v, 'baseline': None, 'ratio': None} 
                      for k, v in current.items()}, args.threshold)
        return 0
    
    if not current:
        print("Error: No current performance data available")
        return 1
    
    # Check for regressions
    has_regression, results = check_regression(baseline, current, args.threshold)
    
    # Print results
    print_results(results, args.threshold)
    
    # Save JSON output if requested
    if args.json_output:
        output_data = {
            'threshold': args.threshold,
            'has_regression': has_regression,
            'results': results,
            'summary': {
                'total_tests': len(results),
                'regressions': sum(1 for r in results.values() if r['status'] == 'regression'),
                'improvements': sum(1 for r in results.values() if r['status'] == 'improved'),
                'stable': sum(1 for r in results.values() if r['status'] == 'stable'),
                'new': sum(1 for r in results.values() if r['status'] == 'new')
            }
        }
        
        with open(args.json_output, 'w') as f:
            json.dump(output_data, f, indent=2)
        print(f"Detailed results saved to {args.json_output}")
    
    # Exit with appropriate code
    if has_regression and args.fail_on_regression:
        print("\n❌ Performance regression detected - failing build")
        return 1
    elif has_regression:
        print("\n⚠️  Performance regression detected - but not failing build")
        return 0
    else:
        print("\n✅ No performance regressions detected")
        return 0


if __name__ == '__main__':
    sys.exit(main())