# WGO Unix-Style Output

WGO now follows Unix philosophy - simple, composable output that works well with other tools.

## Default Output (like `git diff`)

When drift is detected:
```bash
$ wgo diff
--- aws_instance/i-1234567890abcdef0
+++ aws_instance/i-1234567890abcdef0
@@ instance_type @@
-instance_type: t2.micro
+instance_type: t2.small

--- aws_security_group/sg-0123456789abcdef0
+++ aws_security_group/sg-0123456789abcdef0
@@ ingress_rules @@
-ingress_rules: [{"port": 80, "protocol": "tcp"}]
+ingress_rules: [{"port": 80, "protocol": "tcp"}, {"port": 443, "protocol": "tcp"}]

2 additions, 0 deletions, 1 modifications
```

When no drift is detected:
```bash
$ wgo diff
$ echo $?
0
```

## Simple Format (like `git status --short`)

```bash
$ wgo diff --format simple
M aws_instance/i-1234567890abcdef0
M aws_security_group/sg-0123456789abcdef0
```

Where:
- `M` = Modified
- `A` = Added
- `D` = Deleted

## Name Only (like `git diff --name-only`)

```bash
$ wgo diff --name-only
aws_instance/i-1234567890abcdef0
aws_security_group/sg-0123456789abcdef0
```

## Statistics (like `git diff --stat`)

```bash
$ wgo diff --stat
 aws_instance/i-1234567890abcdef0        | 1 change
 aws_security_group/sg-0123456789abcdef0 | 1 change
 2 resources changed, 2 modifications
```

## Quiet Mode (like `git diff --quiet`)

```bash
$ wgo diff --quiet
$ echo $?
1  # Exit code 1 means drift detected, 0 means no drift
```

## Integration with Unix Tools

### Check for any drift
```bash
if wgo diff --quiet; then
    echo "Infrastructure is in sync"
else
    echo "Drift detected!"
fi
```

### Count changed resources
```bash
wgo diff --name-only | wc -l
```

### Filter by resource type
```bash
wgo diff --name-only | grep aws_instance
```

### Generate a report only if drift exists
```bash
wgo diff --quiet || wgo diff > drift-report.txt
```

### CI/CD Pipeline Example
```bash
#!/bin/bash
# Check infrastructure drift in CI

if ! wgo diff --quiet; then
    echo "ERROR: Infrastructure drift detected"
    wgo diff --stat
    exit 1
fi
echo "✓ Infrastructure matches baseline"
```

## Key Design Principles

1. **No output when nothing changed** - Like Unix tools, silence is golden
2. **Exit codes matter** - 0 = no drift, 1 = drift detected
3. **Simple, parseable output** - Easy to pipe to other tools
4. **No scores or risk ratings** - Just facts about what changed
5. **Minimal decoration** - No emojis, boxes, or colors by default

This makes WGO a true "git diff for infrastructure" - simple, reliable, and composable.