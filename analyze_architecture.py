#\!/usr/bin/env python3
"""
VAINO Architecture Analysis - Dependency Violation Detection
Maps out import dependencies and detects architectural violations
"""

import subprocess
import re
import sys
from collections import defaultdict, deque

# Define architectural hierarchy according to VAINO rules
HIERARCHY = {
    'cmd/': 1,                           # Level 1 (Highest)
    'internal/commands/': 2,             # Level 2
    'internal/watchers/': 2,             
    'internal/workers/': 2,              
    'internal/collectors/': 3,           # Level 3
    'internal/storage/': 3,              
    'internal/analysis/': 3,             
    'internal/analyzer/': 3,             # analyzer is same as analysis
    'internal/differ/': 3,
    'internal/scanner/': 3,
    'internal/output/': 3,
    'internal/cache/': 3,
    'internal/discovery/': 3,
    'internal/catchup/': 3,
    'internal/visualization/': 3,
    'internal/app/': 3,
    'internal/config/': 4,               # Level 4
    'internal/errors/': 4,               
    'internal/utils/': 4,                
    'internal/helpers/': 4,              
    'internal/logger/': 4,               
    'internal/ai/': 4,                   # AI module
    'pkg/types/': 5,                     # Level 5 (Lowest)
    'pkg/config/': 5,
    'pkg/collectors/': 5,
}

def get_package_level(package_path):
    """Determine architectural level of a package"""
    for prefix, level in HIERARCHY.items():
        if prefix in package_path:
            return level, prefix
    return None, None

def extract_imports():
    """Extract all internal imports from Go files"""
    try:
        result = subprocess.run(
            ['go', 'list', '-f', '{{.ImportPath}}: {{.Imports}}', './...'],
            capture_output=True, text=True, cwd='/Users/yair/projects/vaino'
        )
        
        if result.returncode \!= 0:
            print(f"Error running go list: {result.stderr}")
            return {}
        
        imports = {}
        for line in result.stdout.strip().split('\n'):
            if not line.strip() or ':' not in line:
                continue
                
            parts = line.split(': ', 1)
            if len(parts) \!= 2:
                continue
                
            package = parts[0].strip()
            import_list = parts[1].strip('[]').split()
            
            # Filter for internal imports only
            internal_imports = [
                imp for imp in import_list 
                if 'github.com/yairfalse/vaino' in imp
            ]
            
            if internal_imports:
                imports[package] = internal_imports
                
        return imports
    except Exception as e:
        print(f"Error extracting imports: {e}")
        return {}

def analyze_violations(imports):
    """Analyze architectural violations"""
    violations = []
    level_violations = []
    pkg_to_internal = []
    
    for package, import_list in imports.items():
        pkg_level, pkg_prefix = get_package_level(package)
        
        # Check pkg importing from internal
        if package.startswith('github.com/yairfalse/vaino/pkg/'):
            for imp in import_list:
                if 'internal/' in imp:
                    pkg_to_internal.append(f"  {package} imports {imp}")
        
        if pkg_level is None:
            continue
            
        for imp in import_list:
            imp_level, imp_prefix = get_package_level(imp)
            
            if imp_level is None:
                continue
                
            # Level violations: higher level importing from lower level
            if pkg_level > imp_level:  # Remember: higher number = lower level
                level_violations.append({
                    'package': package,
                    'import': imp,
                    'pkg_level': pkg_level,
                    'pkg_prefix': pkg_prefix,
                    'imp_level': imp_level,
                    'imp_prefix': imp_prefix,
                    'violation_type': 'upward_dependency'
                })
    
    return level_violations, pkg_to_internal

def find_circular_dependencies(imports):
    """Find circular dependencies using DFS"""
    # Build adjacency list
    graph = defaultdict(set)
    for package, import_list in imports.items():
        for imp in import_list:
            if imp in imports:  # Only consider packages we have import info for
                graph[package].add(imp)
    
    def has_path(start, end, visited=None):
        if visited is None:
            visited = set()
        if start == end:
            return True
        if start in visited:
            return False
        visited.add(start)
        
        for neighbor in graph[start]:
            if has_path(neighbor, end, visited.copy()):
                return True
        return False
    
    cycles = []
    checked = set()
    
    for package in graph:
        if package in checked:
            continue
            
        for imported in graph[package]:
            if imported in graph and has_path(imported, package):
                if (package, imported) not in checked and (imported, package) not in checked:
                    cycles.append((package, imported))
                    checked.add((package, imported))
                    checked.add((imported, package))
    
    return cycles

def print_analysis(imports, violations, pkg_violations, cycles):
    """Print comprehensive architectural analysis"""
    print("=" * 80)
    print("🏗️  VAINO ARCHITECTURAL ANALYSIS")
    print("=" * 80)
    
    # Summary
    print("\n📊 SUMMARY")
    print("-" * 40)
    total_packages = len(imports)
    total_violations = len(violations)
    total_pkg_violations = len(pkg_violations)
    total_cycles = len(cycles)
    
    print(f"Total packages analyzed: {total_packages}")
    print(f"Level violations found: {total_violations}")
    print(f"pkg→internal violations: {total_pkg_violations}")
    print(f"Circular dependencies: {total_cycles}")
    
    overall_status = "🟢 CLEAN" if (total_violations + total_pkg_violations + total_cycles) == 0 else "🔴 VIOLATIONS FOUND"
    print(f"Overall status: {overall_status}")
    
    # Level-based package distribution
    print("\n🏗️  ARCHITECTURAL LEVELS")
    print("-" * 40)
    level_distribution = defaultdict(list)
    for package in imports.keys():
        level, prefix = get_package_level(package)
        if level:
            level_distribution[level].append((package, prefix))
    
    for level in sorted(level_distribution.keys()):
        print(f"Level {level}: {len(level_distribution[level])} packages")
        for package, prefix in sorted(level_distribution[level]):
            short_name = package.replace('github.com/yairfalse/vaino/', '')
            print(f"  {short_name} ({prefix})")
    
    # Architectural violations
    if violations:
        print(f"\n🚨 ARCHITECTURAL LEVEL VIOLATIONS ({len(violations)})")
        print("-" * 60)
        print("Higher-level packages importing from lower-level packages:")
        print("(Remember: Level 1 = highest, Level 5 = lowest)")
        print()
        
        for v in sorted(violations, key=lambda x: (x['pkg_level'], x['imp_level'])):
            pkg_short = v['package'].replace('github.com/yairfalse/vaino/', '')
            imp_short = v['import'].replace('github.com/yairfalse/vaino/', '')
            print(f"❌ {pkg_short} (Level {v['pkg_level']}) → {imp_short} (Level {v['imp_level']})")
            print(f"   {v['pkg_prefix']} should not import from {v['imp_prefix']}")
            print()
    
    # pkg→internal violations
    if pkg_violations:
        print(f"\n🚨 PKG→INTERNAL VIOLATIONS ({len(pkg_violations)})")
        print("-" * 60)
        print("pkg/ packages importing from internal/ packages:")
        for violation in pkg_violations:
            print(f"❌ {violation}")
    
    # Circular dependencies
    if cycles:
        print(f"\n🔄 CIRCULAR DEPENDENCIES ({len(cycles)})")
        print("-" * 60)
        for pkg1, pkg2 in cycles:
            pkg1_short = pkg1.replace('github.com/yairfalse/vaino/', '')
            pkg2_short = pkg2.replace('github.com/yairfalse/vaino/', '')
            print(f"↻ {pkg1_short} ↔ {pkg2_short}")
    
    # Recommendations
    print("\n📋 RECOMMENDATIONS")
    print("-" * 60)
    
    if not violations and not pkg_violations and not cycles:
        print("✅ Architecture is clean\! No violations found.")
        print("✅ All packages respect the hierarchical boundaries.")
        print("✅ No circular dependencies detected.")
    else:
        if violations:
            print("🔧 Level Violations:")
            print("   - Refactor upward dependencies")
            print("   - Move shared code to appropriate level")
            print("   - Consider dependency inversion patterns")
            
        if pkg_violations:
            print("🔧 pkg→internal Violations:")
            print("   - Move shared interfaces to pkg/")
            print("   - Create proper abstraction layers")
            print("   - Ensure pkg/ remains implementation-agnostic")
            
        if cycles:
            print("🔧 Circular Dependencies:")
            print("   - Break cycles with dependency inversion")
            print("   - Extract common interfaces")
            print("   - Use event-driven patterns")
    
    print("\n🎯 NEXT STEPS")
    print("-" * 60)
    if total_violations + total_pkg_violations + total_cycles > 0:
        print("1. Fix architectural violations before adding new features")
        print("2. Implement strict dependency checking in CI/CD")
        print("3. Add architecture tests to prevent regressions")
        print("4. Consider using dependency injection framework")
    else:
        print("1. Add architectural tests to prevent future violations")
        print("2. Document architectural decisions")
        print("3. Enforce rules in pre-commit hooks")
    
    print("\n" + "=" * 80)

def main():
    print("Analyzing VAINO codebase architecture...")
    
    imports = extract_imports()
    if not imports:
        print("❌ No imports found or error occurred")
        return 1
    
    violations, pkg_violations = analyze_violations(imports)
    cycles = find_circular_dependencies(imports)
    
    print_analysis(imports, violations, pkg_violations, cycles)
    
    # Return exit code based on violations
    total_issues = len(violations) + len(pkg_violations) + len(cycles)
    return 1 if total_issues > 0 else 0

if __name__ == "__main__":
    sys.exit(main())
EOF < /dev/null