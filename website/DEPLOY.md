# Deploy to Vercel

## Quick Deploy

```bash
cd website
npm i -g vercel
vercel
```

## Or connect to GitHub

1. Go to [vercel.com](https://vercel.com)
2. Import GitHub repo: `zulfff/FortressWAF`
3. Set root directory to: `website`
4. Deploy

## Environment Variables (if needed)

```
NEXT_PUBLIC_API_URL=https://api.fortresswaf.io
NEXT_PUBLIC_DASHBOARD_URL=https://fortresswaf.io
```

## Custom Domain

1. Go to Vercel Dashboard → Settings → Domains
2. Add: `fortresswaf.io`
3. Add DNS records:
   - A record: `@ 76.76.21.21`
   - CNAME: `www` → `cname.vercel-dns.com`

## After Deploy

- Website: https://fortresswaf.vercel.app
- Or your custom domain
