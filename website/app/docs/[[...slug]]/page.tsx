import defaultMdxComponents from "fumadocs-ui/mdx";
import {
  DocsPage,
  DocsBody,
  DocsTitle,
  DocsDescription,
} from "fumadocs-ui/page";
import { source } from "@/lib/source";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";

interface PageData {
  title: string;
  description?: string;
  full?: boolean;
  body: (props: { components: Record<string, unknown> }) => ReactNode;
  toc: { title: ReactNode; url: string; depth: number }[];
}

export default async function Page(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const data = page.data as unknown as PageData;
  const MDX = data.body;

  return (
    <DocsPage
      toc={data.toc}
      full={data.full}
      editOnGithub={{
        owner: "delavalom",
        repo: "graft",
        sha: "main",
        path: `website/content/docs/${page.path}`,
      }}
    >
      <DocsTitle>{data.title}</DocsTitle>
      <DocsDescription>{data.description}</DocsDescription>
      <DocsBody>
        <MDX components={{ ...defaultMdxComponents }} />
      </DocsBody>
    </DocsPage>
  );
}

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(props: {
  params: Promise<{ slug?: string[] }>;
}) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const data = page.data as unknown as PageData;

  return {
    title: `${data.title} – Graft Docs`,
    description: data.description,
  };
}
