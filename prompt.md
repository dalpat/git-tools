You are working on this repo. Your job is to pick exactly ONE open GitHub issue, implement it, and commit.

1. Fetch all open issues for this repo using `gh issue list --json number,title,body,labels,assignees,linkedPullRequests`. There will a parent issues thats serving as PRD/Tasks DO NOT close that once you are done.
2. Pick the issue YOU think should be done next. Be opinionated:
   - Prefer issues blocked by NOTHING first.
   - Among unblocked issues, pick by priority: critical > high > medium > low.
   - If priorities tie, pick the one YOU judge has the most value or is a prerequisite for others. You Do NOT have to pick the first one.
   - If there are no independant issues left, then pick the issue based on what you think is important. 
3. Implement ONLY that issue. Do not scope-creep.
4. Work in a feature branch, create the branch if needed as `feature/branch-name` format
5. Run your feedback loops: tests, type checks, lint. Fix failures.
6. Append your progress to the progress.txt file, create the file if does not exist.
7. Make a git commit of that feature. ONLY WORK ON A SINGLE FEATURE. Commit with a message that references the issue: `fixes #N` or `refs #N`.
8. If, while implementing the feature, you notice that all work is complete, output <promise>COMPLETE</promise>.
