# Delegating Log Analysis
To prevent context overflow when reading massive server logs, use the delegate command to spawn a subagent.

Example Trigger:
```bash
delegate "Read /var/log/nginx/error.log and summarize the frequency of 502 errors over the last 24 hours. Do not output the raw logs, just the summary statistics."
```
