# Portable pnpm Setup

This project is configured for maximum portability. All dependencies are installed locally without symlinks.

## Configuration Applied

The `.npmrc` file has been configured with:
- `node-linker=hoisted`: Uses a flat node_modules structure like npm
- `symlink=false`: Disables symbolic links
- `store-dir=node_modules/.pnpm-store`: Stores packages locally in the project
- `virtual-store-dir=node_modules/.pnpm`: Virtual store also local
- `enable-global-dir=false`: Disables global directory usage

## Copying to Another PC

When copying this project to another PC:

1. **Copy the entire project folder** including the `node_modules` directory
2. **No additional setup required** - all dependencies are self-contained
3. **pnpm commands will work immediately** without reinstalling

## Verifying Portability

To verify the setup is working correctly:

```bash
# Check that packages are installed locally (not symlinked)
pnpm list

# Run the development server
pnpm dev

# Build the project
pnpm build
```

## File Structure
- `node_modules/`: All packages installed here (not symlinked)
- `node_modules/.pnpm-store/`: Local package store
- `node_modules/.pnpm/`: Virtual store for pnpm
- `.npmrc`: Configuration file ensuring local installation

## Notes
- The project folder will be larger due to local package storage
- No internet connection required on the target PC for existing dependencies
- All binaries and executables are included locally