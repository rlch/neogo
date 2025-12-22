// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightSidebarTopics from 'starlight-sidebar-topics';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'neogo',
			description: 'A Golang ORM for Neo4j which creates idiomatic & fluent Cypher',
			logo: {
				src: './src/assets/logo.png',
				alt: 'neogo',
				replacesTitle: true,
			},
			favicon: '/favicon.svg',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/rlch/neogo' },
			],
			editLink: {
				baseUrl: 'https://github.com/rlch/neogo/edit/main/docs/',
			},
			head: [
				{
					tag: 'meta',
					attrs: {
						property: 'og:image',
						content: 'https://i.imgur.com/4bK7CqC.png',
					},
				},
			],
			plugins: [
				starlightSidebarTopics([
					{
						label: 'Guide',
						link: '/getting-started/introduction/',
						icon: 'open-book',
						items: [
							{
								label: 'Getting Started',
								items: [
									{ label: 'Introduction', slug: 'getting-started/introduction' },
									{ label: 'Installation', slug: 'getting-started/installation' },
									{ label: 'Quick Start', slug: 'getting-started/quick-start' },
								],
							},
							{
								label: 'Declaring Models',
								items: [
									{ label: 'Overview', slug: 'models/overview' },
									{ label: 'Nodes', slug: 'models/nodes' },
									{ label: 'Relationships', slug: 'models/relationships' },
									{ label: 'Abstract Nodes', slug: 'models/abstract-nodes' },
									{ label: 'Labels & Types', slug: 'models/labels' },
									{ label: 'Struct Tags', slug: 'models/tags' },
								],
							},
							{
								label: 'Cypher Queries',
								items: [
									{ label: 'Writing Queries', slug: 'crud/cypher' },
								],
							},
							{
								label: 'Sessions & Transactions',
								items: [
									{ label: 'Sessions', slug: 'transactions/sessions' },
									{ label: 'Transactions', slug: 'transactions/transactions' },
									{ label: 'Streaming Results', slug: 'transactions/streaming' },
								],
							},
							{
								label: 'Schema',
								items: [
									{ label: 'Auto Migration', slug: 'schema/migration' },
									{ label: 'Indexes', slug: 'schema/indexes' },
									{ label: 'Constraints', slug: 'schema/constraints' },
								],
							},
						],
					},
					{
						label: 'Reference',
						link: '/reference/configuration/',
						icon: 'document',
						items: [
							{
								label: 'Reference',
								items: [
									{ label: 'Configuration', slug: 'reference/configuration' },
									{ label: 'Struct Tags', slug: 'reference/struct-tags' },
									{ label: 'API Reference', slug: 'reference/api' },
								],
							},
						],
					},
				]),
			],
			customCss: ['./src/styles/custom.css'],
		}),
	],
});
