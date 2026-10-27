---
name: django-setup
description: Scaffolds a new Django application following the organization's standard architecture.
allowed-tools: [exec, read, write]
---
# Django Setup Procedure
## Context
Use this skill when scaffolding a new Django backend service.

## Steps
1. Run `django-admin startproject config .`
2. Create standard apps: `python manage.py startapp core` and `python manage.py startapp api`
3. Install standard dependencies: `pip install djangorestframework django-cors-headers celery`
4. Update `config/settings.py` to include `rest_framework`, `corsheaders`, `core`, and `api` in `INSTALLED_APPS`.

## Best Practices
- Always configure Celery for async tasks immediately.
- Use PostgreSQL as the default database, never SQLite for production.
