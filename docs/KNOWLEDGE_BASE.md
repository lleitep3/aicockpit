# AICockpit Knowledge Base System

## Overview

The Knowledge Base (KB) system in AICockpit is a powerful tool for organizing, searching, and managing documentation. It combines keyword-based search with semantic search capabilities to help AI agents find relevant information quickly and efficiently.

## Features

- **Keyword Search**: Fast, exact-match search based on document titles, tags, descriptions, and content
- **Semantic Search**: Concept-based search that finds related documents even without exact keyword matches
- **Metadata System**: Structured metadata headers for organizing documents
- **Scoring System**: Probabilistic scoring (0-1) for search results
- **Multiple Output Formats**: JSON, table, and human-readable formats
- **File-Based Storage**: Documents stored as Markdown files with YAML metadata

## Document Format

All KB documents follow a standard format with metadata header and content:

```markdown
---
title: "Document Title"
description: "Brief description of the document"
tags: ["tag1", "tag2", "tag3"]
created: "2026-06-20"
modified: "2026-06-20"
author: "Author Name"
version: "1.0"
related: ["doc-id-1", "doc-id-2"]
---

# Document Content

Your markdown content here...
```

### Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Document title |
| `description` | string | No | Brief description (used in search) |
| `tags` | array | No | Search tags for categorization |
| `created` | date | No | Creation date (auto-set if missing) |
| `modified` | date | No | Last modification date (auto-set if missing) |
| `author` | string | No | Document author |
| `version` | string | No | Document version |
| `related` | array | No | IDs of related documents |

## Directory Structure

Knowledge base documents are organized in `~/.cockpit/kb/`:

```
~/.cockpit/kb/
├── guides/                    # How-to guides and tutorials
├── references/                # Technical references
├── examples/                  # Code examples
├── troubleshooting/           # Problem solutions
└── best-practices/            # Best practices and patterns
```

## Using the KB Command

### Search Documents

```bash
# Basic search
cockpit kb search "logging configuration"

# Search with output format
cockpit kb search "logging" --format json
cockpit kb search "logging" --format table

# Limit results
cockpit kb search "logging" --limit 5
```

### List All Documents

```bash
# List all documents
cockpit kb list

# List in JSON format
cockpit kb list --format json

# List in table format
cockpit kb list --format table
```

### Add Documents

```bash
# Add a new document to KB
cockpit kb add /path/to/document.md
```

### Remove Documents

```bash
# Remove a document by ID
cockpit kb remove document-id
```

## Search Scoring

The KB system uses a sophisticated scoring algorithm to rank search results:

### Keyword Score (40% weight)

Calculated based on:
- **Title Match** (0.5): Presence in document title
- **Tags Match** (0.3): Exact or partial match with tags
- **Description Match** (0.2): Presence in description
- **Content Frequency** (0.1): Keyword density in content

### Semantic Score (60% weight)

- Calculated using embeddings (when available)
- Measures conceptual similarity
- Finds related documents even without exact matches

### Final Score

```
Final Score = (Keyword Score × 0.4) + (Semantic Score × 0.6)
```

## Search Result Format

### Default Format

```
Search Results for: "logging"
Found: 2 documents

1. Logging Configuration Guide
   ID: logging-setup
   Score: 0.92 (keyword: 0.85, semantic: 0.98)
   Tags: [logging, configuration, setup]
   Excerpt: The system of logging can be configured through the config.yaml file...
   Path: guides/logging-setup.md
```

### JSON Format

```json
{
  "query": "logging",
  "results": [
    {
      "id": "logging-setup",
      "title": "Logging Configuration Guide",
      "description": "How to configure logging",
      "path": "guides/logging-setup.md",
      "score": 0.92,
      "keyword_score": 0.85,
      "semantic_score": 0.98,
      "tags": ["logging", "configuration"],
      "excerpt": "...",
      "created": "2026-06-20T00:00:00Z",
      "modified": "2026-06-20T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Table Format

```
Search Results for: "logging"
Found: 2 documents

Title                          | Score | Keywords | Path
Logging Configuration Guide    | 0.92  | 0.85     | guides/logging-setup.md
Troubleshooting Logging Issues | 0.78  | 0.65     | troubleshooting/logging-issues.md
```

## Creating Documents

### Step 1: Create Markdown File

Create a new `.md` file in the appropriate subdirectory:

```bash
touch ~/.cockpit/kb/guides/my-guide.md
```

### Step 2: Add Metadata Header

Start with the metadata header:

```yaml
---
title: "My Guide Title"
description: "Brief description"
tags: ["tag1", "tag2"]
author: "Your Name"
version: "1.0"
---
```

### Step 3: Add Content

Write your content in Markdown:

```markdown
# My Guide Title

## Introduction

Explain what this guide covers...

## Getting Started

Step-by-step instructions...

## Examples

Code examples and use cases...
```

### Step 4: Verify

Search for your document:

```bash
cockpit kb search "my guide"
```

## Best Practices

1. **Use Clear Titles**: Make titles descriptive and searchable
2. **Add Tags**: Use 3-5 relevant tags for better categorization
3. **Write Descriptions**: Keep descriptions under 100 characters
4. **Link Related Docs**: Use the `related` field to connect documents
5. **Keep Content Updated**: Update the `modified` date when making changes
6. **Use Consistent Formatting**: Follow Markdown conventions
7. **Organize by Type**: Place documents in appropriate subdirectories

## Examples

### Example 1: Configuration Guide

```markdown
---
title: "Logging Configuration Guide"
description: "How to configure logging in AICockpit"
tags: ["logging", "configuration", "setup"]
author: "AICockpit Team"
version: "1.0"
related: ["troubleshooting-logging-issues"]
---

# Logging Configuration Guide

## Overview

AICockpit provides a comprehensive logging system...

## Configuration

### Basic Setup

Edit `~/.cockpit/config.yaml`:

```yaml
log_level: "info"
```

### Log Levels

- **debug**: Detailed information
- **info**: General information
- **warn**: Warning messages
- **error**: Error messages
```

### Example 2: Troubleshooting Guide

```markdown
---
title: "Troubleshooting Logging Issues"
description: "Solutions for common logging problems"
tags: ["logging", "troubleshooting", "debugging"]
author: "AICockpit Team"
version: "1.0"
related: ["logging-setup"]
---

# Troubleshooting Logging Issues

## Problem: Logs Not Being Created

### Solution

1. Check if logs directory exists
2. Run setup if needed
3. Verify permissions
```

## Integration with AI Agents

The KB system is designed to integrate seamlessly with AI agents:

1. **Skill**: Use the `kb-search` skill to search the knowledge base
2. **Hook**: Automatic KB search can be triggered on specific events
3. **Context**: Search results are passed as context to AI agents

See [Skills](./SKILLS.md) and [Hooks](./HOOKS.md) for more information.

## Troubleshooting

### No Documents Found

1. Check KB directory exists: `ls -la ~/.cockpit/kb/`
2. Verify documents have `.md` extension
3. Check metadata header format

### Search Returns No Results

1. Try simpler search terms
2. Check document tags and titles
3. Use `cockpit kb list` to verify documents exist

### Incorrect Scoring

1. Verify document metadata is complete
2. Check keyword relevance to content
3. Review tag accuracy

## Performance

- **Search Speed**: < 100ms for typical KB (< 1000 documents)
- **Memory Usage**: Minimal (documents loaded on demand)
- **Scalability**: Tested with 1000+ documents

## Future Enhancements

- Semantic search with embeddings
- Full-text indexing
- Document versioning
- Collaborative editing
- Integration with external knowledge sources
