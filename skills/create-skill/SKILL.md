---
name: create-skill
description: Use when the user wants to create, write or improve a skill (SKILL.md) that teaches NØX a task.
---

# Create a skill

A skill teaches you one kind of task. You see only its name and the first ~110 characters of its description in your prompt; you read the full SKILL.md when a task matches. So the description decides whether the skill is ever used, and the body must be short enough to follow.

## 1. Understand the task

Ask the user, briefly, only what you cannot infer:
- What task should it handle? Get 2–3 example requests in their words.
- What does a good result look like? Any fixed steps, tools, commands, file locations, style rules?
- All projects or only this one?

## 2. Pick the place and name

- All projects: `~/.nox-local/skills/<name>/SKILL.md`
- One project: `<project>/.nox/skills/<name>/SKILL.md`
- `<name>`: lowercase-with-hyphens, says what it does (`czech-invoice`, `release-notes`). Check it is free: `ls ~/.nox-local/skills .nox/skills 2>/dev/null`

## 3. Write SKILL.md

```markdown
---
name: <name>
description: Use when <trigger in the user's words>; <what it produces>.
---

# <Title>

<One sentence: what this skill does and when.>

## Steps
1. <Concrete action: a command, a file to read, a check.>
2. …

## Rules
- <Things to always / never do.>

## Example
<One short example request and what you would do.>
```

Rules for a good skill:
- **Description:** starts with "Use when", names the situations and keywords a user would actually type, under 110 characters. One line, no line breaks.
- **Body:** under ~80 lines. Numbered steps with exact commands and paths, not general advice. Small local models follow short, concrete lists best.
- Long reference material (tables, templates, API notes) goes in separate files next to SKILL.md, e.g. `reference.md` or `templates/`, and the steps say when to read them.
- Helper scripts go in `scripts/`. Make them safe to run twice, and never destructive without asking.
- Write in English unless the user wants another language; the user's examples can stay in their language.

## 4. Check it

1. Read the file back once and fix anything vague.
2. Tell the user the path and give one example request that should trigger the skill. It is listed from the next message on.
3. Offer to test it: the user sends the example request and you follow the skill exactly. Fix the steps that went wrong.

## Improving an existing skill

Read it, ask what went wrong when it was used, then make the smallest change that fixes it, usually a clearer step or a better description. Keep the user's wording where it works.
