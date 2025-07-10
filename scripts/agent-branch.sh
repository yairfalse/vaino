#!/bin/bash

# Agent Branch Management System for VAINO
# Enables multiple claude-code instances to work simultaneously without conflicts

set -euo pipefail

# Configuration
AGENT_WORK_DIR=".agent-work"
AGENT_CONFIG_FILE="$AGENT_WORK_DIR/config.json"
TASK_DIR="$AGENT_WORK_DIR/tasks"
AGENT_DIR="$AGENT_WORK_DIR/agents"
CONFLICT_DIR="$AGENT_WORK_DIR/conflicts"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Generate unique agent ID
generate_agent_id() {
    local timestamp=$(date +%s)
    local random=$(shuf -i 1000-9999 -n 1)
    echo "agent-${timestamp}-${random}"
}

# Generate task ID
generate_task_id() {
    local timestamp=$(date +%s)
    local random=$(shuf -i 100-999 -n 1)
    echo "task-${timestamp}-${random}"
}

# Initialize agent work directory
init_agent_work() {
    log_info "Initializing agent work directory..."
    
    mkdir -p "$AGENT_WORK_DIR"
    mkdir -p "$TASK_DIR"
    mkdir -p "$AGENT_DIR"
    mkdir -p "$CONFLICT_DIR"
    
    # Create config file if it doesn't exist
    if [[ ! -f "$AGENT_CONFIG_FILE" ]]; then
        cat > "$AGENT_CONFIG_FILE" << EOF
{
  "version": "1.0.0",
  "created": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "max_concurrent_agents": 5,
  "branch_prefix": "feature/agent-",
  "task_timeout_minutes": 60,
  "conflict_detection": true
}
EOF
    fi
    
    log_success "Agent work directory initialized"
}

# Check if agent system is initialized
check_initialized() {
    if [[ ! -d "$AGENT_WORK_DIR" ]]; then
        log_error "Agent system not initialized. Run: $0 init"
        exit 1
    fi
}

# Register new agent
register_agent() {
    local agent_id="$1"
    local task_description="$2"
    
    log_info "Registering agent $agent_id..."
    
    local agent_file="$AGENT_DIR/$agent_id.json"
    cat > "$agent_file" << EOF
{
  "id": "$agent_id",
  "status": "active",
  "created": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "last_activity": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "task_description": "$task_description",
  "branch": "",
  "files_claimed": [],
  "pid": $$
}
EOF
    
    log_success "Agent $agent_id registered"
}

# Create task branch
create_task_branch() {
    local agent_id="$1"
    local component="$2"
    local action="$3"
    
    # Sanitize component and action for branch name
    component=$(echo "$component" | sed 's/[^a-zA-Z0-9]/-/g' | tr '[:upper:]' '[:lower:]')
    action=$(echo "$action" | sed 's/[^a-zA-Z0-9]/-/g' | tr '[:upper:]' '[:lower:]')
    
    local branch_name="feature/$agent_id/$component-$action"
    
    log_info "Creating branch: $branch_name"
    
    # Ensure we're on main branch
    git checkout main
    git pull origin main
    
    # Create and checkout new branch
    git checkout -b "$branch_name"
    
    # Update agent file with branch info
    local agent_file="$AGENT_DIR/$agent_id.json"
    if [[ -f "$agent_file" ]]; then
        jq --arg branch "$branch_name" '.branch = $branch | .last_activity = now | .last_activity |= todate' "$agent_file" > "$agent_file.tmp" && mv "$agent_file.tmp" "$agent_file"
    fi
    
    log_success "Branch $branch_name created and checked out"
    echo "$branch_name"
}

# Create task
create_task() {
    local agent_id="$1"
    local task_description="$2"
    local component="$3"
    local action="$4"
    local files="$5"
    
    local task_id=$(generate_task_id)
    local task_file="$TASK_DIR/$task_id.json"
    
    log_info "Creating task: $task_id"
    
    # Convert files string to JSON array
    local files_json="[]"
    if [[ -n "$files" ]]; then
        files_json=$(echo "$files" | jq -R 'split(",") | map(gsub("^\\s+|\\s+$"; ""))')
    fi
    
    cat > "$task_file" << EOF
{
  "id": "$task_id",
  "agent_id": "$agent_id",
  "description": "$task_description",
  "component": "$component",
  "action": "$action",
  "files": $files_json,
  "status": "active",
  "created": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "started": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "estimated_duration_minutes": 30,
  "priority": "medium"
}
EOF
    
    log_success "Task $task_id created"
    echo "$task_id"
}

# Check for file conflicts
check_file_conflicts() {
    local files="$1"
    local current_agent="$2"
    
    log_info "Checking for file conflicts..."
    
    local conflicts=false
    
    # Check each file against all active agents
    for file in $(echo "$files" | tr ',' '\n'); do
        file=$(echo "$file" | xargs) # trim whitespace
        
        for agent_file in "$AGENT_DIR"/*.json; do
            if [[ -f "$agent_file" ]]; then
                local agent_id=$(jq -r '.id' "$agent_file")
                local agent_status=$(jq -r '.status' "$agent_file")
                
                # Skip if same agent or inactive agent
                if [[ "$agent_id" == "$current_agent" || "$agent_status" != "active" ]]; then
                    continue
                fi
                
                # Check if file is claimed by another agent
                local claimed=$(jq -r --arg file "$file" '.files_claimed[] | select(. == $file)' "$agent_file")
                if [[ -n "$claimed" ]]; then
                    log_warning "File conflict detected: $file is claimed by $agent_id"
                    conflicts=true
                    
                    # Log conflict
                    local conflict_file="$CONFLICT_DIR/conflict-$(date +%s).json"
                    cat > "$conflict_file" << EOF
{
  "file": "$file",
  "agent1": "$current_agent",
  "agent2": "$agent_id",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
  "resolved": false
}
EOF
                fi
            fi
        done
    done
    
    if [[ "$conflicts" == "true" ]]; then
        log_error "File conflicts detected. Please resolve conflicts or choose different files."
        return 1
    fi
    
    log_success "No file conflicts detected"
    return 0
}

# Claim files for agent
claim_files() {
    local agent_id="$1"
    local files="$2"
    
    log_info "Claiming files for agent $agent_id..."
    
    local agent_file="$AGENT_DIR/$agent_id.json"
    if [[ -f "$agent_file" ]]; then
        # Convert files string to JSON array and update agent file
        local files_json="[]"
        if [[ -n "$files" ]]; then
            files_json=$(echo "$files" | jq -R 'split(",") | map(gsub("^\\s+|\\s+$"; ""))')
        fi
        
        jq --argjson files "$files_json" '.files_claimed = $files | .last_activity = now | .last_activity |= todate' "$agent_file" > "$agent_file.tmp" && mv "$agent_file.tmp" "$agent_file"
        
        log_success "Files claimed successfully"
    else
        log_error "Agent file not found: $agent_file"
        return 1
    fi
}

# Show agent status
show_status() {
    log_info "Agent Branch Management System Status"
    echo "======================================"
    
    if [[ ! -d "$AGENT_WORK_DIR" ]]; then
        log_warning "Agent system not initialized"
        return 0
    fi
    
    echo -e "\n${BLUE}Active Agents:${NC}"
    local active_count=0
    
    for agent_file in "$AGENT_DIR"/*.json; do
        if [[ -f "$agent_file" ]]; then
            local agent_id=$(jq -r '.id' "$agent_file")
            local status=$(jq -r '.status' "$agent_file")
            local branch=$(jq -r '.branch' "$agent_file")
            local task_desc=$(jq -r '.task_description' "$agent_file")
            local files=$(jq -r '.files_claimed[]' "$agent_file" 2>/dev/null | tr '\n' ',' | sed 's/,$//')
            
            if [[ "$status" == "active" ]]; then
                echo "  • $agent_id"
                echo "    Branch: $branch"
                echo "    Task: $task_desc"
                if [[ -n "$files" ]]; then
                    echo "    Files: $files"
                fi
                echo
                ((active_count++))
            fi
        fi
    done
    
    if [[ $active_count -eq 0 ]]; then
        echo "  No active agents"
    fi
    
    echo -e "\n${BLUE}Recent Tasks:${NC}"
    local task_count=0
    
    for task_file in "$TASK_DIR"/*.json; do
        if [[ -f "$task_file" ]]; then
            local task_id=$(jq -r '.id' "$task_file")
            local agent_id=$(jq -r '.agent_id' "$task_file")
            local description=$(jq -r '.description' "$task_file")
            local status=$(jq -r '.status' "$task_file")
            local created=$(jq -r '.created' "$task_file")
            
            echo "  • $task_id ($agent_id)"
            echo "    Description: $description"
            echo "    Status: $status"
            echo "    Created: $created"
            echo
            ((task_count++))
            
            # Show only last 5 tasks
            if [[ $task_count -ge 5 ]]; then
                break
            fi
        fi
    done
    
    if [[ $task_count -eq 0 ]]; then
        echo "  No tasks found"
    fi
    
    # Show conflicts if any
    if [[ -n "$(ls -A "$CONFLICT_DIR" 2>/dev/null)" ]]; then
        echo -e "\n${YELLOW}Conflicts:${NC}"
        for conflict_file in "$CONFLICT_DIR"/*.json; do
            if [[ -f "$conflict_file" ]]; then
                local file=$(jq -r '.file' "$conflict_file")
                local agent1=$(jq -r '.agent1' "$conflict_file")
                local agent2=$(jq -r '.agent2' "$conflict_file")
                local resolved=$(jq -r '.resolved' "$conflict_file")
                
                if [[ "$resolved" == "false" ]]; then
                    echo "  • $file: $agent1 vs $agent2"
                fi
            fi
        done
    fi
}

# Cleanup finished agents
cleanup_agents() {
    log_info "Cleaning up finished agents..."
    
    local cleaned=0
    
    for agent_file in "$AGENT_DIR"/*.json; do
        if [[ -f "$agent_file" ]]; then
            local agent_id=$(jq -r '.id' "$agent_file")
            local status=$(jq -r '.status' "$agent_file")
            local pid=$(jq -r '.pid' "$agent_file")
            
            # Check if process is still running
            if [[ "$status" == "active" ]] && ! kill -0 "$pid" 2>/dev/null; then
                log_info "Cleaning up inactive agent: $agent_id"
                jq '.status = "inactive" | .last_activity = now | .last_activity |= todate' "$agent_file" > "$agent_file.tmp" && mv "$agent_file.tmp" "$agent_file"
                ((cleaned++))
            fi
        fi
    done
    
    log_success "Cleaned up $cleaned agents"
}

# Interactive agent start
interactive_start() {
    log_info "Starting interactive agent creation..."
    
    echo "Agent Branch Management System"
    echo "=============================="
    echo
    
    # Get task description
    read -p "Enter task description: " task_description
    if [[ -z "$task_description" ]]; then
        log_error "Task description cannot be empty"
        exit 1
    fi
    
    # Get component
    read -p "Enter component (e.g., docs, collectors, watchers): " component
    if [[ -z "$component" ]]; then
        log_error "Component cannot be empty"
        exit 1
    fi
    
    # Get action
    read -p "Enter action (e.g., refactor, add, fix): " action
    if [[ -z "$action" ]]; then
        log_error "Action cannot be empty"
        exit 1
    fi
    
    # Get files (optional)
    read -p "Enter files to work on (comma-separated, optional): " files
    
    # Generate agent ID
    local agent_id=$(generate_agent_id)
    
    # Initialize if needed
    if [[ ! -d "$AGENT_WORK_DIR" ]]; then
        init_agent_work
    fi
    
    # Check for conflicts if files specified
    if [[ -n "$files" ]]; then
        if ! check_file_conflicts "$files" "$agent_id"; then
            exit 1
        fi
    fi
    
    # Register agent
    register_agent "$agent_id" "$task_description"
    
    # Create task
    local task_id=$(create_task "$agent_id" "$task_description" "$component" "$action" "$files")
    
    # Create branch
    local branch_name=$(create_task_branch "$agent_id" "$component" "$action")
    
    # Claim files
    if [[ -n "$files" ]]; then
        claim_files "$agent_id" "$files"
    fi
    
    echo
    log_success "Agent setup complete!"
    echo "Agent ID: $agent_id"
    echo "Task ID: $task_id"
    echo "Branch: $branch_name"
    echo "Task: $task_description"
    if [[ -n "$files" ]]; then
        echo "Files: $files"
    fi
    echo
    echo "Next steps:"
    echo "1. Make your changes"
    echo "2. Run: make agent-check"
    echo "3. Run: make pr-ready"
    echo "4. Create PR with: gh pr create"
}

# Main function
main() {
    case "${1:-help}" in
        "init")
            init_agent_work
            ;;
        "start")
            interactive_start
            ;;
        "status")
            show_status
            ;;
        "cleanup")
            cleanup_agents
            ;;
        "help"|*)
            echo "Agent Branch Management System for VAINO"
            echo "Usage: $0 [command]"
            echo
            echo "Commands:"
            echo "  init     Initialize agent work directory"
            echo "  start    Start interactive agent creation"
            echo "  status   Show current agent status"
            echo "  cleanup  Clean up finished agents"
            echo "  help     Show this help message"
            echo
            echo "Makefile targets:"
            echo "  make agent-start    Start interactive agent creation"
            echo "  make agent-status   Show agent status"
            echo "  make agent-check    Run quality checks"
            echo "  make pr-ready       Prepare for PR creation"
            ;;
    esac
}

# Run main function
main "$@"