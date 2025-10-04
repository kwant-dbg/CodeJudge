This workflow automatically deploys the contents of `maintenance-page/` to the `gh-pages` branch whenever files under that folder are pushed to `main` or `master`.

To configure a custom domain:

- Add a `CNAME` file under `maintenance-page/` with your domain (for example `www.example.com`). The workflow will copy it into the published content.
- Ensure your DNS is pointed to GitHub Pages (A records for apex or CNAME for subdomains).

If you prefer manual control, you can disable the workflow and push to `gh-pages` yourself.
