# Working with Git Worktrees in Scion

Scion leverages [Git Worktrees](https://git-scm.com/docs/git-worktree) to provide isolated workspaces for agents. This allows each agent to work on its own branch simultaneously without interfering with your main working directory or other agents.

## Prerequisites

To use Scion's worktree features, you must have a modern version of Git installed:

- **Required Version**: Git **2.47.0** or newer.
- **Reason**: Scion uses the `--relative-paths` flag when creating worktrees to ensure that worktree metadata remains valid when mounted in a container. This feature was stabilized in recent Git versions.

## How Scion Uses Worktrees

### Automatic Activation

Scion detects if your current project is a Git repository.
- **Inside a Git Repo**: Scion automatically creates a git worktree for the agent's workspace.
- **Outside a Git Repo**: Scion simply creates a standard directory for the workspace.

This behavior is automatic and requires no configuration.

### Worktree Location

When an agent is created, its files are stored in `agents/<agent-name>/`.
- The **Home Directory** (`agents/<agent-name>/home/`) contains configuration and persistent files.
- The **Workspace** (`agents/<agent-name>/workspace/`) is the actual git worktree.

### Branching Strategy

When Scion creates a worktree, it determines which branch to check out based on the following logic:

1. **New Branch (Default)**:
   If no specific branch is requested, Scion creates a new branch named after the agent (e.g., `my-agent`).
   - This branch is created starting from your **current HEAD**.
   - **Branch-of-Branch**: If you are currently on a feature branch (e.g., `feature/login`), the agent's branch will be based on `feature/login`, effectively creating a "branch of a branch". This is useful for agents helping with specific tasks on an existing feature.

2. **Existing Branch**:
   If you specify a branch (e.g., via configuration or flags), Scion will attempt to check out that existing branch into the worktree.

### Relative Paths

Scion always uses the `--relative-paths` flag:
```bash
git worktree add --relative-paths -b <branch-name> <path>
```
This ensures that the link between the main repository and the worktree uses relative paths in the `.git` files. This makes the entire project directory (including agents) portable to the containerized environments.

## The `cdw` Command

Scion provides a convenient command, `cdw` (Change Directory to Worktree), to quickly navigate to an agent's workspace.

```bash
scion cdw <agent-name>
```

- This command resolves the path to the agent's worktree.
- It spawns a new shell inside that directory.
- It works for both agent names and raw branch names if a corresponding worktree exists.

## Cleanup

When you delete an agent:
```bash
scion delete <agent-name>
```
- The worktree directory is removed.
- `git worktree prune` is called to clean up git metadata.
- **Optional**: You can pass the `-b` or `--remove-branch` flag to also delete the associated git branch.

## Manual Management

While Scion manages these worktrees, they are standard git worktrees. You can list them using standard git commands:

```bash
git worktree list
```

You will see your main working directory and one entry for each active agent.
