import { defineConfig } from "vitepress";

// Project site served at https://opentdm.github.io/opentdm/ — note the base path.
export default defineConfig({
  title: "opentdm",
  description: "Open-source, self-hosted test data & configuration management.",
  base: "/opentdm/",
  lastUpdated: true,
  cleanUrls: true,
  themeConfig: {
    nav: [
      { text: "Guide", link: "/guide/introduction" },
      { text: "Reference", link: "/reference/rest-api" },
      { text: "Changelog", link: "/reference/changelog" },
      { text: "v0.1.0", link: "/reference/changelog" },
    ],
    sidebar: {
      "/guide/": [
        {
          text: "Getting started",
          items: [
            { text: "Introduction", link: "/guide/introduction" },
            { text: "Quickstart (self-host)", link: "/guide/quickstart" },
            { text: "Configuration", link: "/guide/configuration" },
          ],
        },
        {
          text: "Using opentdm",
          items: [
            { text: "Access control", link: "/guide/access-control" },
            { text: "Audit log", link: "/guide/audit-log" },
            { text: "In CI", link: "/guide/ci" },
            { text: "CLI", link: "/guide/cli" },
          ],
        },
        {
          text: "Concepts",
          items: [
            { text: "Architecture", link: "/guide/architecture" },
            { text: "Security", link: "/guide/security" },
          ],
        },
      ],
      "/reference/": [
        {
          text: "Reference",
          items: [
            { text: "REST API", link: "/reference/rest-api" },
            { text: "Releasing", link: "/reference/releasing" },
            { text: "Contributing", link: "/reference/contributing" },
            { text: "Changelog", link: "/reference/changelog" },
          ],
        },
      ],
    },
    socialLinks: [{ icon: "github", link: "https://github.com/opentdm/opentdm" }],
    search: { provider: "local" },
    editLink: {
      pattern: "https://github.com/opentdm/opentdm/edit/main/docs/:path",
      text: "Edit this page on GitHub",
    },
    footer: {
      message: "Released under the MIT License.",
      copyright: "© opentdm contributors",
    },
  },
});
