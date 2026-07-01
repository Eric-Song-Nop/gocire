import { siteData } from "../generated/site-data";

export const prerender = true;

type SitePage = {
  href: string;
  date: string;
};

const ymdPattern = /^(\d{4})-(\d{2})-(\d{2})$/;

export async function GET() {
  const siteUrl = requiredSiteUrl();
  const urls = sitePages().map((page) => renderUrl(siteUrl, page)).join("");
  const xml = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    urls.trimEnd(),
    "</urlset>",
    "",
  ]
    .filter((line) => line !== "")
    .join("\n");

  return new Response(`${xml}\n`, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
    },
  });
}

function requiredSiteUrl() {
  const siteUrl = cleanText(siteData.site.url);
  if (!siteUrl) {
    throw new Error("site.url is required to generate sitemap.xml");
  }
  return siteUrl;
}

function sitePages() {
  return siteData.pages as readonly SitePage[];
}

function renderUrl(siteUrl: string, page: SitePage) {
  const lines = [
    "  <url>",
    `    <loc>${escapeXml(joinSiteUrl(siteUrl, page.href))}</loc>`,
  ];

  const lastmod = validYmdDate(page.date);
  if (lastmod) {
    lines.push(`    <lastmod>${escapeXml(lastmod)}</lastmod>`);
  }

  lines.push("  </url>");
  return `${lines.join("\n")}\n`;
}

function validYmdDate(value: string) {
  const date = cleanText(value);
  const ymd = ymdPattern.exec(date);
  if (!ymd) {
    return "";
  }

  const year = Number(ymd[1]);
  const month = Number(ymd[2]);
  const day = Number(ymd[3]);
  const timestamp = Date.UTC(year, month - 1, day);
  const parsed = new Date(timestamp);
  if (parsed.getUTCFullYear() !== year || parsed.getUTCMonth() !== month - 1 || parsed.getUTCDate() !== day) {
    return "";
  }
  return date;
}

function joinSiteUrl(siteUrl: string, href: string) {
  const base = siteUrl.replace(/\/+$/, "");
  const path = cleanText(href).replace(/^\/+/, "");
  return path ? `${base}/${path}` : `${base}/`;
}

function escapeXml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

function cleanText(value: unknown) {
  return String(value ?? "").trim();
}
