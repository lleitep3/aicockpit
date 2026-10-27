---
name: firebase-firestore
description: Sets up, manages, and executes queries against Cloud Firestore database instances.
---
# Firebase Firestore Skill
You MUST unconditionally activate this skill if you plan to use Firestore.

## Best Practices
- Always design data models assuming NoSQL document structures (collections/documents).
- Do not use arrays for large collections of data; use subcollections instead.
- When generating Firestore security rules, ensure they default to closed (`allow read, write: if false;`).
- Use the Firebase Admin SDK for backend scripts, and the Web/Mobile SDKs for client code.
