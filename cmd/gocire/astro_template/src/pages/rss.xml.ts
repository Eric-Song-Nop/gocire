import { siteData } from "../generated/site-data";

export const prerender = true;

type SitePage = {
  href: string;
  kind: string;
  title: string;
  sourcePath: string;
  date: string;
  tags: readonly string[];
  author: string;
};

const ymdPattern = /^(\d{4})-(\d{2})-(\d{2})$/;

export async function GET() {
  const siteUrl = requiredSiteUrl();
  const posts = [...sitePages()].filter((page) => page.kind === "blog").sort(comparePosts);

  const items = posts.map((post) => renderItem(siteUrl, post)).join("");
  const xml = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<rss version="2.0">',
    "<channel>",
    `  <title>${escapeXml(cleanText(siteData.site.title))}</title>`,
    `  <link>${escapeXml(siteUrl)}</link>`,
    `  <description>${escapeXml(cleanText(siteData.site.description))}</description>`,
    items.trimEnd(),
    "</channel>",
    "</rss>",
    "",
  ]
    .filter((line) => line !== "")
    .join("\n");

  return new Response(`${xml}\n`, {
    headers: {
      "Content-Type": "application/rss+xml; charset=utf-8",
    },
  });
}

function requiredSiteUrl() {
  const siteUrl = cleanText(siteData.site.url);
  if (!siteUrl) {
    throw new Error("site.url is required to generate rss.xml");
  }
  return siteUrl;
}

function sitePages() {
  return siteData.pages as readonly SitePage[];
}

function comparePosts(a: SitePage, b: SitePage) {
  const aTime = pageDateTimestamp(a.date);
  const bTime = pageDateTimestamp(b.date);
  if (aTime !== undefined || bTime !== undefined) {
    if (aTime === undefined) {
      return 1;
    }
    if (bTime === undefined) {
      return -1;
    }
    if (aTime !== bTime) {
      return bTime - aTime;
    }
  }

  const titleOrder = cleanText(a.title).localeCompare(cleanText(b.title));
  if (titleOrder !== 0) {
    return titleOrder;
  }
  return cleanText(a.href).localeCompare(cleanText(b.href));
}

function renderItem(siteUrl: string, post: SitePage) {
  const link = joinSiteUrl(siteUrl, post.href);
  const title = cleanText(post.title) || cleanText(post.sourcePath) || cleanText(post.href);
  const lines = [
    "  <item>",
    `    <title>${escapeXml(title)}</title>`,
    `    <link>${escapeXml(link)}</link>`,
    `    <guid>${escapeXml(link)}</guid>`,
  ];

  const pubDate = rssPubDate(post.date);
  if (pubDate) {
    lines.push(`    <pubDate>${escapeXml(pubDate)}</pubDate>`);
  }

  for (const tag of post.tags) {
    const category = cleanText(tag);
    if (category) {
      lines.push(`    <category>${escapeXml(category)}</category>`);
    }
  }

  const author = cleanText(post.author);
  if (author) {
    lines.push(`    <author>${escapeXml(author)}</author>`);
  }

  lines.push("  </item>");
  return `${lines.join("\n")}\n`;
}

function rssPubDate(value: string) {
  const timestamp = pageDateTimestamp(value);
  return timestamp === undefined ? "" : new Date(timestamp).toUTCString();
}

function pageDateTimestamp(value: string) {
  const date = cleanText(value);
  if (!date) {
    return undefined;
  }

  const ymd = ymdPattern.exec(date);
  if (ymd) {
    const year = Number(ymd[1]);
    const month = Number(ymd[2]);
    const day = Number(ymd[3]);
    const timestamp = Date.UTC(year, month - 1, day);
    const parsed = new Date(timestamp);
    if (parsed.getUTCFullYear() !== year || parsed.getUTCMonth() !== month - 1 || parsed.getUTCDate() !== day) {
      return undefined;
    }
    return timestamp;
  }

  const timestamp = Date.parse(date);
  return Number.isNaN(timestamp) ? undefined : timestamp;
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
