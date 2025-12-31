# Seed Data Files

This directory contains SQL seed files for populating database branches with test data.

## Usage

Seed files are automatically executed when creating a branch with `--clone-data seed_data`:

```bash
# Use default seeds from this directory
fluxbase branch create my-branch --clone-data seed_data

# Use seeds from a custom directory
fluxbase branch create my-branch --clone-data seed_data --seeds-dir ./custom-seeds
```

## File Naming Convention

- Files must have a `.sql` extension
- Use numeric prefixes for execution order: `001_`, `002_`, `003_`, etc.
- Files execute in lexicographic order

## Best Practices

1. **Make seeds idempotent** - Use `ON CONFLICT DO NOTHING` or check for existence before inserting
2. **Use deterministic IDs** - Hardcode UUIDs for test data so they're consistent across environments
3. **Keep files focused** - One file per logical group (users, posts, etc.)
4. **Add comments** - Document what each seed file does and any dependencies
5. **Avoid sensitive data** - Never commit real user data, passwords, or API keys

## Example Files

- `001_example_users.sql` - Example test users
- More examples can be added as needed

## Configuration

Configure the default seeds directory in `fluxbase.yaml`:

```yaml
branching:
  enabled: true
  seeds_path: ./seeds  # Path to this directory
```

## Troubleshooting

If seed execution fails:

1. Check the branch status: `fluxbase branch get <branch-name>`
2. View activity log: `fluxbase branch activity <branch-name>`
3. Fix the seed file
4. Reset the branch: `fluxbase branch reset <branch-name> --force`

The branch database is kept even if seeding fails, so you can investigate the issue.
