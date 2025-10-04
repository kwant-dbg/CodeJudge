# Maintenance page

This folder contains a small static maintenance page you can publish with GitHub Pages and point your website domain to while your main site is offline to save Azure Student credits.

Files:

- `index.html` — the maintenance page HTML.
- `styles.css` — simple styling.

How to publish (two easy options):

1) Publish from a branch using GitHub Pages (recommended)

- Create a new branch (for example `gh-pages`) and copy the contents of this folder to the repository root or to a `docs/` folder.
- Commit and push the branch.
- In your repository settings -> Pages, select the branch (e.g., `gh-pages` or `main` with `/docs` folder) as the source.

2) Use the `docs/` folder on the default branch

- Move these files into the `docs/` directory on your `main` (or `master`) branch and enable GitHub Pages from the branch root with `/docs` as source.

Adding a custom domain (CNAME)

- Create a file named `CNAME` (no extension) in the same folder that is served (root or `docs/`) containing your domain, e.g.: `www.example.com`.
- On your DNS provider, point your domain to GitHub Pages:
  - For apex/root domain (example.com): create A records pointing to GitHub Pages IP addresses (185.199.108.153, 185.199.109.153, 185.199.110.153, 185.199.111.153).
  - For www (subdomain): create a CNAME record pointing to `<your-github-username>.github.io`.

Notes and tips

- GitHub Pages serves over HTTPS — GitHub will manage TLS automatically.
- Using `gh-pages` branch is convenient: you can deploy only this static page without changing your main site.
- If you want automated deployment, use GitHub Actions to copy `maintenance-page/` to `gh-pages` on push to a selected branch.

If you want, I can: create the `gh-pages` branch and open a PR that deploys this page, or add a GitHub Action that publishes to `gh-pages` automatically. Tell me which option you prefer and your repository's GitHub username (if different from the repo owner) and custom domain (if you want me to pre-fill `CNAME`).
 
Automatic deployment helper
---------------------------

This repository includes a small GitHub Actions workflow `.github/workflows/deploy-maintenance-on-change.yml` that will automatically publish the contents of `maintenance-page/` to the `gh-pages` branch whenever files under that folder change on `main` or `master`. The workflow is lightweight and only runs on maintenance-page changes (so it won't trigger the full CI).

How it works:
- Edit files under `maintenance-page/` on `main` or `master` and push.
- The workflow will run, copy `maintenance-page/` into an `out/` folder and deploy `out/` to `gh-pages` using `peaceiris/actions-gh-pages`.
- The `gh-pages` branch will be overwritten with the deployed content (this is intentional to keep Pages content minimal).

If you prefer manual control, simply push to the existing `gh-pages` branch instead.
