declare module 'astro:content' {
	interface Render {
		'.mdx': Promise<{
			Content: import('astro').MarkdownInstance<{}>['Content'];
			headings: import('astro').MarkdownHeading[];
			remarkPluginFrontmatter: Record<string, any>;
			components: import('astro').MDXInstance<{}>['components'];
		}>;
	}
}

declare module 'astro:content' {
	interface RenderResult {
		Content: import('astro/runtime/server/index.js').AstroComponentFactory;
		headings: import('astro').MarkdownHeading[];
		remarkPluginFrontmatter: Record<string, any>;
	}
	interface Render {
		'.md': Promise<RenderResult>;
	}

	export interface RenderedContent {
		html: string;
		metadata?: {
			imagePaths: Array<string>;
			[key: string]: unknown;
		};
	}
}

declare module 'astro:content' {
	type Flatten<T> = T extends { [K: string]: infer U } ? U : never;

	export type CollectionKey = keyof AnyEntryMap;
	export type CollectionEntry<C extends CollectionKey> = Flatten<AnyEntryMap[C]>;

	export type ContentCollectionKey = keyof ContentEntryMap;
	export type DataCollectionKey = keyof DataEntryMap;

	type AllValuesOf<T> = T extends any ? T[keyof T] : never;
	type ValidContentEntrySlug<C extends keyof ContentEntryMap> = AllValuesOf<
		ContentEntryMap[C]
	>['slug'];

	/** @deprecated Use `getEntry` instead. */
	export function getEntryBySlug<
		C extends keyof ContentEntryMap,
		E extends ValidContentEntrySlug<C> | (string & {}),
	>(
		collection: C,
		// Note that this has to accept a regular string too, for SSR
		entrySlug: E,
	): E extends ValidContentEntrySlug<C>
		? Promise<CollectionEntry<C>>
		: Promise<CollectionEntry<C> | undefined>;

	/** @deprecated Use `getEntry` instead. */
	export function getDataEntryById<C extends keyof DataEntryMap, E extends keyof DataEntryMap[C]>(
		collection: C,
		entryId: E,
	): Promise<CollectionEntry<C>>;

	export function getCollection<C extends keyof AnyEntryMap, E extends CollectionEntry<C>>(
		collection: C,
		filter?: (entry: CollectionEntry<C>) => entry is E,
	): Promise<E[]>;
	export function getCollection<C extends keyof AnyEntryMap>(
		collection: C,
		filter?: (entry: CollectionEntry<C>) => unknown,
	): Promise<CollectionEntry<C>[]>;

	export function getEntry<
		C extends keyof ContentEntryMap,
		E extends ValidContentEntrySlug<C> | (string & {}),
	>(entry: {
		collection: C;
		slug: E;
	}): E extends ValidContentEntrySlug<C>
		? Promise<CollectionEntry<C>>
		: Promise<CollectionEntry<C> | undefined>;
	export function getEntry<
		C extends keyof DataEntryMap,
		E extends keyof DataEntryMap[C] | (string & {}),
	>(entry: {
		collection: C;
		id: E;
	}): E extends keyof DataEntryMap[C]
		? Promise<DataEntryMap[C][E]>
		: Promise<CollectionEntry<C> | undefined>;
	export function getEntry<
		C extends keyof ContentEntryMap,
		E extends ValidContentEntrySlug<C> | (string & {}),
	>(
		collection: C,
		slug: E,
	): E extends ValidContentEntrySlug<C>
		? Promise<CollectionEntry<C>>
		: Promise<CollectionEntry<C> | undefined>;
	export function getEntry<
		C extends keyof DataEntryMap,
		E extends keyof DataEntryMap[C] | (string & {}),
	>(
		collection: C,
		id: E,
	): E extends keyof DataEntryMap[C]
		? Promise<DataEntryMap[C][E]>
		: Promise<CollectionEntry<C> | undefined>;

	/** Resolve an array of entry references from the same collection */
	export function getEntries<C extends keyof ContentEntryMap>(
		entries: {
			collection: C;
			slug: ValidContentEntrySlug<C>;
		}[],
	): Promise<CollectionEntry<C>[]>;
	export function getEntries<C extends keyof DataEntryMap>(
		entries: {
			collection: C;
			id: keyof DataEntryMap[C];
		}[],
	): Promise<CollectionEntry<C>[]>;

	export function render<C extends keyof AnyEntryMap>(
		entry: AnyEntryMap[C][string],
	): Promise<RenderResult>;

	export function reference<C extends keyof AnyEntryMap>(
		collection: C,
	): import('astro/zod').ZodEffects<
		import('astro/zod').ZodString,
		C extends keyof ContentEntryMap
			? {
					collection: C;
					slug: ValidContentEntrySlug<C>;
				}
			: {
					collection: C;
					id: keyof DataEntryMap[C];
				}
	>;
	// Allow generic `string` to avoid excessive type errors in the config
	// if `dev` is not running to update as you edit.
	// Invalid collection names will be caught at build time.
	export function reference<C extends string>(
		collection: C,
	): import('astro/zod').ZodEffects<import('astro/zod').ZodString, never>;

	type ReturnTypeOrOriginal<T> = T extends (...args: any[]) => infer R ? R : T;
	type InferEntrySchema<C extends keyof AnyEntryMap> = import('astro/zod').infer<
		ReturnTypeOrOriginal<Required<ContentConfig['collections'][C]>['schema']>
	>;

	type ContentEntryMap = {
		"docs": {
"api-cookbook.md": {
	id: "api-cookbook.md";
  slug: "api-cookbook";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/http/index.md": {
	id: "api/http/index.md";
  slug: "api/http";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/README.md": {
	id: "api/sdk-react/README.md";
  slug: "api/sdk-react/readme";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/classes/FluxbaseClient.md": {
	id: "api/sdk-react/classes/FluxbaseClient.md";
  slug: "api/sdk-react/classes/fluxbaseclient";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/FluxbaseProvider.md": {
	id: "api/sdk-react/functions/FluxbaseProvider.md";
  slug: "api/sdk-react/functions/fluxbaseprovider";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useAPIKeys.md": {
	id: "api/sdk-react/functions/useAPIKeys.md";
  slug: "api/sdk-react/functions/useapikeys";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useAdminAuth.md": {
	id: "api/sdk-react/functions/useAdminAuth.md";
  slug: "api/sdk-react/functions/useadminauth";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useAppSettings.md": {
	id: "api/sdk-react/functions/useAppSettings.md";
  slug: "api/sdk-react/functions/useappsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useAuth.md": {
	id: "api/sdk-react/functions/useAuth.md";
  slug: "api/sdk-react/functions/useauth";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useCreateBucket.md": {
	id: "api/sdk-react/functions/useCreateBucket.md";
  slug: "api/sdk-react/functions/usecreatebucket";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useDelete.md": {
	id: "api/sdk-react/functions/useDelete.md";
  slug: "api/sdk-react/functions/usedelete";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useDeleteBucket.md": {
	id: "api/sdk-react/functions/useDeleteBucket.md";
  slug: "api/sdk-react/functions/usedeletebucket";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useFluxbaseClient.md": {
	id: "api/sdk-react/functions/useFluxbaseClient.md";
  slug: "api/sdk-react/functions/usefluxbaseclient";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useFluxbaseQuery.md": {
	id: "api/sdk-react/functions/useFluxbaseQuery.md";
  slug: "api/sdk-react/functions/usefluxbasequery";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useInsert.md": {
	id: "api/sdk-react/functions/useInsert.md";
  slug: "api/sdk-react/functions/useinsert";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useRPC.md": {
	id: "api/sdk-react/functions/useRPC.md";
  slug: "api/sdk-react/functions/userpc";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useRPCBatch.md": {
	id: "api/sdk-react/functions/useRPCBatch.md";
  slug: "api/sdk-react/functions/userpcbatch";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useRPCMutation.md": {
	id: "api/sdk-react/functions/useRPCMutation.md";
  slug: "api/sdk-react/functions/userpcmutation";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useRealtime.md": {
	id: "api/sdk-react/functions/useRealtime.md";
  slug: "api/sdk-react/functions/userealtime";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useSession.md": {
	id: "api/sdk-react/functions/useSession.md";
  slug: "api/sdk-react/functions/usesession";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useSignIn.md": {
	id: "api/sdk-react/functions/useSignIn.md";
  slug: "api/sdk-react/functions/usesignin";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useSignOut.md": {
	id: "api/sdk-react/functions/useSignOut.md";
  slug: "api/sdk-react/functions/usesignout";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useSignUp.md": {
	id: "api/sdk-react/functions/useSignUp.md";
  slug: "api/sdk-react/functions/usesignup";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageBuckets.md": {
	id: "api/sdk-react/functions/useStorageBuckets.md";
  slug: "api/sdk-react/functions/usestoragebuckets";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageCopy.md": {
	id: "api/sdk-react/functions/useStorageCopy.md";
  slug: "api/sdk-react/functions/usestoragecopy";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageDelete.md": {
	id: "api/sdk-react/functions/useStorageDelete.md";
  slug: "api/sdk-react/functions/usestoragedelete";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageDownload.md": {
	id: "api/sdk-react/functions/useStorageDownload.md";
  slug: "api/sdk-react/functions/usestoragedownload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageList.md": {
	id: "api/sdk-react/functions/useStorageList.md";
  slug: "api/sdk-react/functions/usestoragelist";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageMove.md": {
	id: "api/sdk-react/functions/useStorageMove.md";
  slug: "api/sdk-react/functions/usestoragemove";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStoragePublicUrl.md": {
	id: "api/sdk-react/functions/useStoragePublicUrl.md";
  slug: "api/sdk-react/functions/usestoragepublicurl";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageSignedUrl.md": {
	id: "api/sdk-react/functions/useStorageSignedUrl.md";
  slug: "api/sdk-react/functions/usestoragesignedurl";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useStorageUpload.md": {
	id: "api/sdk-react/functions/useStorageUpload.md";
  slug: "api/sdk-react/functions/usestorageupload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useSystemSettings.md": {
	id: "api/sdk-react/functions/useSystemSettings.md";
  slug: "api/sdk-react/functions/usesystemsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useTable.md": {
	id: "api/sdk-react/functions/useTable.md";
  slug: "api/sdk-react/functions/usetable";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useTableDeletes.md": {
	id: "api/sdk-react/functions/useTableDeletes.md";
  slug: "api/sdk-react/functions/usetabledeletes";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useTableInserts.md": {
	id: "api/sdk-react/functions/useTableInserts.md";
  slug: "api/sdk-react/functions/usetableinserts";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useTableSubscription.md": {
	id: "api/sdk-react/functions/useTableSubscription.md";
  slug: "api/sdk-react/functions/usetablesubscription";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useTableUpdates.md": {
	id: "api/sdk-react/functions/useTableUpdates.md";
  slug: "api/sdk-react/functions/usetableupdates";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useUpdate.md": {
	id: "api/sdk-react/functions/useUpdate.md";
  slug: "api/sdk-react/functions/useupdate";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useUpdateUser.md": {
	id: "api/sdk-react/functions/useUpdateUser.md";
  slug: "api/sdk-react/functions/useupdateuser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useUpsert.md": {
	id: "api/sdk-react/functions/useUpsert.md";
  slug: "api/sdk-react/functions/useupsert";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useUser.md": {
	id: "api/sdk-react/functions/useUser.md";
  slug: "api/sdk-react/functions/useuser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useUsers.md": {
	id: "api/sdk-react/functions/useUsers.md";
  slug: "api/sdk-react/functions/useusers";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/functions/useWebhooks.md": {
	id: "api/sdk-react/functions/useWebhooks.md";
  slug: "api/sdk-react/functions/usewebhooks";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/APIKey.md": {
	id: "api/sdk-react/interfaces/APIKey.md";
  slug: "api/sdk-react/interfaces/apikey";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/AdminUser.md": {
	id: "api/sdk-react/interfaces/AdminUser.md";
  slug: "api/sdk-react/interfaces/adminuser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/AppSettings.md": {
	id: "api/sdk-react/interfaces/AppSettings.md";
  slug: "api/sdk-react/interfaces/appsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/AuthSession.md": {
	id: "api/sdk-react/interfaces/AuthSession.md";
  slug: "api/sdk-react/interfaces/authsession";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/EnrichedUser.md": {
	id: "api/sdk-react/interfaces/EnrichedUser.md";
  slug: "api/sdk-react/interfaces/enricheduser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/PostgrestResponse.md": {
	id: "api/sdk-react/interfaces/PostgrestResponse.md";
  slug: "api/sdk-react/interfaces/postgrestresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/RealtimeChangePayload.md": {
	id: "api/sdk-react/interfaces/RealtimeChangePayload.md";
  slug: "api/sdk-react/interfaces/realtimechangepayload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/SignInCredentials.md": {
	id: "api/sdk-react/interfaces/SignInCredentials.md";
  slug: "api/sdk-react/interfaces/signincredentials";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/SignUpCredentials.md": {
	id: "api/sdk-react/interfaces/SignUpCredentials.md";
  slug: "api/sdk-react/interfaces/signupcredentials";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/SystemSetting.md": {
	id: "api/sdk-react/interfaces/SystemSetting.md";
  slug: "api/sdk-react/interfaces/systemsetting";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/User.md": {
	id: "api/sdk-react/interfaces/User.md";
  slug: "api/sdk-react/interfaces/user";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/interfaces/Webhook.md": {
	id: "api/sdk-react/interfaces/Webhook.md";
  slug: "api/sdk-react/interfaces/webhook";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk-react/type-aliases/StorageObject.md": {
	id: "api/sdk-react/type-aliases/StorageObject.md";
  slug: "api/sdk-react/type-aliases/storageobject";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/README.md": {
	id: "api/sdk/README.md";
  slug: "api/sdk/readme";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/APIKeysManager.md": {
	id: "api/sdk/classes/APIKeysManager.md";
  slug: "api/sdk/classes/apikeysmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/AppSettingsManager.md": {
	id: "api/sdk/classes/AppSettingsManager.md";
  slug: "api/sdk/classes/appsettingsmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/AuthSettingsManager.md": {
	id: "api/sdk/classes/AuthSettingsManager.md";
  slug: "api/sdk/classes/authsettingsmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/DDLManager.md": {
	id: "api/sdk/classes/DDLManager.md";
  slug: "api/sdk/classes/ddlmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/EmailTemplateManager.md": {
	id: "api/sdk/classes/EmailTemplateManager.md";
  slug: "api/sdk/classes/emailtemplatemanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseAdmin.md": {
	id: "api/sdk/classes/FluxbaseAdmin.md";
  slug: "api/sdk/classes/fluxbaseadmin";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseAdminFunctions.md": {
	id: "api/sdk/classes/FluxbaseAdminFunctions.md";
  slug: "api/sdk/classes/fluxbaseadminfunctions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseAdminJobs.md": {
	id: "api/sdk/classes/FluxbaseAdminJobs.md";
  slug: "api/sdk/classes/fluxbaseadminjobs";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseAdminMigrations.md": {
	id: "api/sdk/classes/FluxbaseAdminMigrations.md";
  slug: "api/sdk/classes/fluxbaseadminmigrations";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseAuth.md": {
	id: "api/sdk/classes/FluxbaseAuth.md";
  slug: "api/sdk/classes/fluxbaseauth";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseClient.md": {
	id: "api/sdk/classes/FluxbaseClient.md";
  slug: "api/sdk/classes/fluxbaseclient";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseFetch.md": {
	id: "api/sdk/classes/FluxbaseFetch.md";
  slug: "api/sdk/classes/fluxbasefetch";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseFunctions.md": {
	id: "api/sdk/classes/FluxbaseFunctions.md";
  slug: "api/sdk/classes/fluxbasefunctions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseJobs.md": {
	id: "api/sdk/classes/FluxbaseJobs.md";
  slug: "api/sdk/classes/fluxbasejobs";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseManagement.md": {
	id: "api/sdk/classes/FluxbaseManagement.md";
  slug: "api/sdk/classes/fluxbasemanagement";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseOAuth.md": {
	id: "api/sdk/classes/FluxbaseOAuth.md";
  slug: "api/sdk/classes/fluxbaseoauth";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseRealtime.md": {
	id: "api/sdk/classes/FluxbaseRealtime.md";
  slug: "api/sdk/classes/fluxbaserealtime";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseSettings.md": {
	id: "api/sdk/classes/FluxbaseSettings.md";
  slug: "api/sdk/classes/fluxbasesettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/FluxbaseStorage.md": {
	id: "api/sdk/classes/FluxbaseStorage.md";
  slug: "api/sdk/classes/fluxbasestorage";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/ImpersonationManager.md": {
	id: "api/sdk/classes/ImpersonationManager.md";
  slug: "api/sdk/classes/impersonationmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/InvitationsManager.md": {
	id: "api/sdk/classes/InvitationsManager.md";
  slug: "api/sdk/classes/invitationsmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/OAuthProviderManager.md": {
	id: "api/sdk/classes/OAuthProviderManager.md";
  slug: "api/sdk/classes/oauthprovidermanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/QueryBuilder.md": {
	id: "api/sdk/classes/QueryBuilder.md";
  slug: "api/sdk/classes/querybuilder";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/RealtimeChannel.md": {
	id: "api/sdk/classes/RealtimeChannel.md";
  slug: "api/sdk/classes/realtimechannel";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/SchemaQueryBuilder.md": {
	id: "api/sdk/classes/SchemaQueryBuilder.md";
  slug: "api/sdk/classes/schemaquerybuilder";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/SettingsClient.md": {
	id: "api/sdk/classes/SettingsClient.md";
  slug: "api/sdk/classes/settingsclient";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/StorageBucket.md": {
	id: "api/sdk/classes/StorageBucket.md";
  slug: "api/sdk/classes/storagebucket";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/SystemSettingsManager.md": {
	id: "api/sdk/classes/SystemSettingsManager.md";
  slug: "api/sdk/classes/systemsettingsmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/classes/WebhooksManager.md": {
	id: "api/sdk/classes/WebhooksManager.md";
  slug: "api/sdk/classes/webhooksmanager";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/functions/createClient.md": {
	id: "api/sdk/functions/createClient.md";
  slug: "api/sdk/functions/createclient";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/APIKey.md": {
	id: "api/sdk/interfaces/APIKey.md";
  slug: "api/sdk/interfaces/apikey";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AcceptInvitationRequest.md": {
	id: "api/sdk/interfaces/AcceptInvitationRequest.md";
  slug: "api/sdk/interfaces/acceptinvitationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AcceptInvitationResponse.md": {
	id: "api/sdk/interfaces/AcceptInvitationResponse.md";
  slug: "api/sdk/interfaces/acceptinvitationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminAuthResponse.md": {
	id: "api/sdk/interfaces/AdminAuthResponse.md";
  slug: "api/sdk/interfaces/adminauthresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminLoginRequest.md": {
	id: "api/sdk/interfaces/AdminLoginRequest.md";
  slug: "api/sdk/interfaces/adminloginrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminMeResponse.md": {
	id: "api/sdk/interfaces/AdminMeResponse.md";
  slug: "api/sdk/interfaces/adminmeresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminRefreshRequest.md": {
	id: "api/sdk/interfaces/AdminRefreshRequest.md";
  slug: "api/sdk/interfaces/adminrefreshrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminRefreshResponse.md": {
	id: "api/sdk/interfaces/AdminRefreshResponse.md";
  slug: "api/sdk/interfaces/adminrefreshresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminSetupRequest.md": {
	id: "api/sdk/interfaces/AdminSetupRequest.md";
  slug: "api/sdk/interfaces/adminsetuprequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminSetupStatusResponse.md": {
	id: "api/sdk/interfaces/AdminSetupStatusResponse.md";
  slug: "api/sdk/interfaces/adminsetupstatusresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AdminUser.md": {
	id: "api/sdk/interfaces/AdminUser.md";
  slug: "api/sdk/interfaces/adminuser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AppSettings.md": {
	id: "api/sdk/interfaces/AppSettings.md";
  slug: "api/sdk/interfaces/appsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ApplyMigrationRequest.md": {
	id: "api/sdk/interfaces/ApplyMigrationRequest.md";
  slug: "api/sdk/interfaces/applymigrationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ApplyPendingRequest.md": {
	id: "api/sdk/interfaces/ApplyPendingRequest.md";
  slug: "api/sdk/interfaces/applypendingrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AuthResponse.md": {
	id: "api/sdk/interfaces/AuthResponse.md";
  slug: "api/sdk/interfaces/authresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AuthSession.md": {
	id: "api/sdk/interfaces/AuthSession.md";
  slug: "api/sdk/interfaces/authsession";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AuthSettings.md": {
	id: "api/sdk/interfaces/AuthSettings.md";
  slug: "api/sdk/interfaces/authsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/AuthenticationSettings.md": {
	id: "api/sdk/interfaces/AuthenticationSettings.md";
  slug: "api/sdk/interfaces/authenticationsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/BroadcastMessage.md": {
	id: "api/sdk/interfaces/BroadcastMessage.md";
  slug: "api/sdk/interfaces/broadcastmessage";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/BundleOptions.md": {
	id: "api/sdk/interfaces/BundleOptions.md";
  slug: "api/sdk/interfaces/bundleoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/BundleResult.md": {
	id: "api/sdk/interfaces/BundleResult.md";
  slug: "api/sdk/interfaces/bundleresult";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Column.md": {
	id: "api/sdk/interfaces/Column.md";
  slug: "api/sdk/interfaces/column";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateAPIKeyRequest.md": {
	id: "api/sdk/interfaces/CreateAPIKeyRequest.md";
  slug: "api/sdk/interfaces/createapikeyrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateAPIKeyResponse.md": {
	id: "api/sdk/interfaces/CreateAPIKeyResponse.md";
  slug: "api/sdk/interfaces/createapikeyresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateColumnRequest.md": {
	id: "api/sdk/interfaces/CreateColumnRequest.md";
  slug: "api/sdk/interfaces/createcolumnrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateFunctionRequest.md": {
	id: "api/sdk/interfaces/CreateFunctionRequest.md";
  slug: "api/sdk/interfaces/createfunctionrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateInvitationRequest.md": {
	id: "api/sdk/interfaces/CreateInvitationRequest.md";
  slug: "api/sdk/interfaces/createinvitationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateInvitationResponse.md": {
	id: "api/sdk/interfaces/CreateInvitationResponse.md";
  slug: "api/sdk/interfaces/createinvitationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateMigrationRequest.md": {
	id: "api/sdk/interfaces/CreateMigrationRequest.md";
  slug: "api/sdk/interfaces/createmigrationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateOAuthProviderRequest.md": {
	id: "api/sdk/interfaces/CreateOAuthProviderRequest.md";
  slug: "api/sdk/interfaces/createoauthproviderrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateOAuthProviderResponse.md": {
	id: "api/sdk/interfaces/CreateOAuthProviderResponse.md";
  slug: "api/sdk/interfaces/createoauthproviderresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateSchemaRequest.md": {
	id: "api/sdk/interfaces/CreateSchemaRequest.md";
  slug: "api/sdk/interfaces/createschemarequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateSchemaResponse.md": {
	id: "api/sdk/interfaces/CreateSchemaResponse.md";
  slug: "api/sdk/interfaces/createschemaresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateTableRequest.md": {
	id: "api/sdk/interfaces/CreateTableRequest.md";
  slug: "api/sdk/interfaces/createtablerequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateTableResponse.md": {
	id: "api/sdk/interfaces/CreateTableResponse.md";
  slug: "api/sdk/interfaces/createtableresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/CreateWebhookRequest.md": {
	id: "api/sdk/interfaces/CreateWebhookRequest.md";
  slug: "api/sdk/interfaces/createwebhookrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DeleteAPIKeyResponse.md": {
	id: "api/sdk/interfaces/DeleteAPIKeyResponse.md";
  slug: "api/sdk/interfaces/deleteapikeyresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DeleteOAuthProviderResponse.md": {
	id: "api/sdk/interfaces/DeleteOAuthProviderResponse.md";
  slug: "api/sdk/interfaces/deleteoauthproviderresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DeleteTableResponse.md": {
	id: "api/sdk/interfaces/DeleteTableResponse.md";
  slug: "api/sdk/interfaces/deletetableresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DeleteUserResponse.md": {
	id: "api/sdk/interfaces/DeleteUserResponse.md";
  slug: "api/sdk/interfaces/deleteuserresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DeleteWebhookResponse.md": {
	id: "api/sdk/interfaces/DeleteWebhookResponse.md";
  slug: "api/sdk/interfaces/deletewebhookresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DownloadOptions.md": {
	id: "api/sdk/interfaces/DownloadOptions.md";
  slug: "api/sdk/interfaces/downloadoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/DownloadProgress.md": {
	id: "api/sdk/interfaces/DownloadProgress.md";
  slug: "api/sdk/interfaces/downloadprogress";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/EdgeFunction.md": {
	id: "api/sdk/interfaces/EdgeFunction.md";
  slug: "api/sdk/interfaces/edgefunction";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/EdgeFunctionExecution.md": {
	id: "api/sdk/interfaces/EdgeFunctionExecution.md";
  slug: "api/sdk/interfaces/edgefunctionexecution";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/EmailSettings.md": {
	id: "api/sdk/interfaces/EmailSettings.md";
  slug: "api/sdk/interfaces/emailsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/EmailTemplate.md": {
	id: "api/sdk/interfaces/EmailTemplate.md";
  slug: "api/sdk/interfaces/emailtemplate";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/EnrichedUser.md": {
	id: "api/sdk/interfaces/EnrichedUser.md";
  slug: "api/sdk/interfaces/enricheduser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FeatureSettings.md": {
	id: "api/sdk/interfaces/FeatureSettings.md";
  slug: "api/sdk/interfaces/featuresettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FileObject.md": {
	id: "api/sdk/interfaces/FileObject.md";
  slug: "api/sdk/interfaces/fileobject";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FluxbaseClientOptions.md": {
	id: "api/sdk/interfaces/FluxbaseClientOptions.md";
  slug: "api/sdk/interfaces/fluxbaseclientoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FluxbaseError.md": {
	id: "api/sdk/interfaces/FluxbaseError.md";
  slug: "api/sdk/interfaces/fluxbaseerror";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FunctionInvokeOptions.md": {
	id: "api/sdk/interfaces/FunctionInvokeOptions.md";
  slug: "api/sdk/interfaces/functioninvokeoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/FunctionSpec.md": {
	id: "api/sdk/interfaces/FunctionSpec.md";
  slug: "api/sdk/interfaces/functionspec";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/GetImpersonationResponse.md": {
	id: "api/sdk/interfaces/GetImpersonationResponse.md";
  slug: "api/sdk/interfaces/getimpersonationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ImpersonateAnonRequest.md": {
	id: "api/sdk/interfaces/ImpersonateAnonRequest.md";
  slug: "api/sdk/interfaces/impersonateanonrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ImpersonateServiceRequest.md": {
	id: "api/sdk/interfaces/ImpersonateServiceRequest.md";
  slug: "api/sdk/interfaces/impersonateservicerequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ImpersonateUserRequest.md": {
	id: "api/sdk/interfaces/ImpersonateUserRequest.md";
  slug: "api/sdk/interfaces/impersonateuserrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ImpersonationSession.md": {
	id: "api/sdk/interfaces/ImpersonationSession.md";
  slug: "api/sdk/interfaces/impersonationsession";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ImpersonationTargetUser.md": {
	id: "api/sdk/interfaces/ImpersonationTargetUser.md";
  slug: "api/sdk/interfaces/impersonationtargetuser";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Invitation.md": {
	id: "api/sdk/interfaces/Invitation.md";
  slug: "api/sdk/interfaces/invitation";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/InviteUserRequest.md": {
	id: "api/sdk/interfaces/InviteUserRequest.md";
  slug: "api/sdk/interfaces/inviteuserrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/InviteUserResponse.md": {
	id: "api/sdk/interfaces/InviteUserResponse.md";
  slug: "api/sdk/interfaces/inviteuserresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListAPIKeysResponse.md": {
	id: "api/sdk/interfaces/ListAPIKeysResponse.md";
  slug: "api/sdk/interfaces/listapikeysresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListEmailTemplatesResponse.md": {
	id: "api/sdk/interfaces/ListEmailTemplatesResponse.md";
  slug: "api/sdk/interfaces/listemailtemplatesresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListImpersonationSessionsOptions.md": {
	id: "api/sdk/interfaces/ListImpersonationSessionsOptions.md";
  slug: "api/sdk/interfaces/listimpersonationsessionsoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListImpersonationSessionsResponse.md": {
	id: "api/sdk/interfaces/ListImpersonationSessionsResponse.md";
  slug: "api/sdk/interfaces/listimpersonationsessionsresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListInvitationsOptions.md": {
	id: "api/sdk/interfaces/ListInvitationsOptions.md";
  slug: "api/sdk/interfaces/listinvitationsoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListInvitationsResponse.md": {
	id: "api/sdk/interfaces/ListInvitationsResponse.md";
  slug: "api/sdk/interfaces/listinvitationsresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListOAuthProvidersResponse.md": {
	id: "api/sdk/interfaces/ListOAuthProvidersResponse.md";
  slug: "api/sdk/interfaces/listoauthprovidersresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListOptions.md": {
	id: "api/sdk/interfaces/ListOptions.md";
  slug: "api/sdk/interfaces/listoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListSchemasResponse.md": {
	id: "api/sdk/interfaces/ListSchemasResponse.md";
  slug: "api/sdk/interfaces/listschemasresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListSystemSettingsResponse.md": {
	id: "api/sdk/interfaces/ListSystemSettingsResponse.md";
  slug: "api/sdk/interfaces/listsystemsettingsresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListTablesResponse.md": {
	id: "api/sdk/interfaces/ListTablesResponse.md";
  slug: "api/sdk/interfaces/listtablesresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListUsersOptions.md": {
	id: "api/sdk/interfaces/ListUsersOptions.md";
  slug: "api/sdk/interfaces/listusersoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListUsersResponse.md": {
	id: "api/sdk/interfaces/ListUsersResponse.md";
  slug: "api/sdk/interfaces/listusersresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListWebhookDeliveriesResponse.md": {
	id: "api/sdk/interfaces/ListWebhookDeliveriesResponse.md";
  slug: "api/sdk/interfaces/listwebhookdeliveriesresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ListWebhooksResponse.md": {
	id: "api/sdk/interfaces/ListWebhooksResponse.md";
  slug: "api/sdk/interfaces/listwebhooksresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/MailgunSettings.md": {
	id: "api/sdk/interfaces/MailgunSettings.md";
  slug: "api/sdk/interfaces/mailgunsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Migration.md": {
	id: "api/sdk/interfaces/Migration.md";
  slug: "api/sdk/interfaces/migration";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/MigrationExecution.md": {
	id: "api/sdk/interfaces/MigrationExecution.md";
  slug: "api/sdk/interfaces/migrationexecution";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/OAuthProvider.md": {
	id: "api/sdk/interfaces/OAuthProvider.md";
  slug: "api/sdk/interfaces/oauthprovider";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/OrderBy.md": {
	id: "api/sdk/interfaces/OrderBy.md";
  slug: "api/sdk/interfaces/orderby";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/PostgresChangesConfig.md": {
	id: "api/sdk/interfaces/PostgresChangesConfig.md";
  slug: "api/sdk/interfaces/postgreschangesconfig";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/PostgrestError.md": {
	id: "api/sdk/interfaces/PostgrestError.md";
  slug: "api/sdk/interfaces/postgresterror";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/PostgrestResponse.md": {
	id: "api/sdk/interfaces/PostgrestResponse.md";
  slug: "api/sdk/interfaces/postgrestresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/PresenceState.md": {
	id: "api/sdk/interfaces/PresenceState.md";
  slug: "api/sdk/interfaces/presencestate";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/QueryFilter.md": {
	id: "api/sdk/interfaces/QueryFilter.md";
  slug: "api/sdk/interfaces/queryfilter";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimeBroadcastPayload.md": {
	id: "api/sdk/interfaces/RealtimeBroadcastPayload.md";
  slug: "api/sdk/interfaces/realtimebroadcastpayload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimeChangePayload.md": {
	id: "api/sdk/interfaces/RealtimeChangePayload.md";
  slug: "api/sdk/interfaces/realtimechangepayload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimeChannelConfig.md": {
	id: "api/sdk/interfaces/RealtimeChannelConfig.md";
  slug: "api/sdk/interfaces/realtimechannelconfig";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimeMessage.md": {
	id: "api/sdk/interfaces/RealtimeMessage.md";
  slug: "api/sdk/interfaces/realtimemessage";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimePostgresChangesPayload.md": {
	id: "api/sdk/interfaces/RealtimePostgresChangesPayload.md";
  slug: "api/sdk/interfaces/realtimepostgreschangespayload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RealtimePresencePayload.md": {
	id: "api/sdk/interfaces/RealtimePresencePayload.md";
  slug: "api/sdk/interfaces/realtimepresencepayload";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RequestOptions.md": {
	id: "api/sdk/interfaces/RequestOptions.md";
  slug: "api/sdk/interfaces/requestoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ResetUserPasswordResponse.md": {
	id: "api/sdk/interfaces/ResetUserPasswordResponse.md";
  slug: "api/sdk/interfaces/resetuserpasswordresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ResumableDownloadData.md": {
	id: "api/sdk/interfaces/ResumableDownloadData.md";
  slug: "api/sdk/interfaces/resumabledownloaddata";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ResumableDownloadOptions.md": {
	id: "api/sdk/interfaces/ResumableDownloadOptions.md";
  slug: "api/sdk/interfaces/resumabledownloadoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RevokeAPIKeyResponse.md": {
	id: "api/sdk/interfaces/RevokeAPIKeyResponse.md";
  slug: "api/sdk/interfaces/revokeapikeyresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RevokeInvitationResponse.md": {
	id: "api/sdk/interfaces/RevokeInvitationResponse.md";
  slug: "api/sdk/interfaces/revokeinvitationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/RollbackMigrationRequest.md": {
	id: "api/sdk/interfaces/RollbackMigrationRequest.md";
  slug: "api/sdk/interfaces/rollbackmigrationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SESSettings.md": {
	id: "api/sdk/interfaces/SESSettings.md";
  slug: "api/sdk/interfaces/sessettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SMTPSettings.md": {
	id: "api/sdk/interfaces/SMTPSettings.md";
  slug: "api/sdk/interfaces/smtpsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Schema.md": {
	id: "api/sdk/interfaces/Schema.md";
  slug: "api/sdk/interfaces/schema";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SecuritySettings.md": {
	id: "api/sdk/interfaces/SecuritySettings.md";
  slug: "api/sdk/interfaces/securitysettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SendGridSettings.md": {
	id: "api/sdk/interfaces/SendGridSettings.md";
  slug: "api/sdk/interfaces/sendgridsettings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SignInCredentials.md": {
	id: "api/sdk/interfaces/SignInCredentials.md";
  slug: "api/sdk/interfaces/signincredentials";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SignInWith2FAResponse.md": {
	id: "api/sdk/interfaces/SignInWith2FAResponse.md";
  slug: "api/sdk/interfaces/signinwith2faresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SignUpCredentials.md": {
	id: "api/sdk/interfaces/SignUpCredentials.md";
  slug: "api/sdk/interfaces/signupcredentials";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SignedUrlOptions.md": {
	id: "api/sdk/interfaces/SignedUrlOptions.md";
  slug: "api/sdk/interfaces/signedurloptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/StartImpersonationResponse.md": {
	id: "api/sdk/interfaces/StartImpersonationResponse.md";
  slug: "api/sdk/interfaces/startimpersonationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/StopImpersonationResponse.md": {
	id: "api/sdk/interfaces/StopImpersonationResponse.md";
  slug: "api/sdk/interfaces/stopimpersonationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/StreamDownloadData.md": {
	id: "api/sdk/interfaces/StreamDownloadData.md";
  slug: "api/sdk/interfaces/streamdownloaddata";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SyncError.md": {
	id: "api/sdk/interfaces/SyncError.md";
  slug: "api/sdk/interfaces/syncerror";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SyncFunctionsOptions.md": {
	id: "api/sdk/interfaces/SyncFunctionsOptions.md";
  slug: "api/sdk/interfaces/syncfunctionsoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SyncFunctionsResult.md": {
	id: "api/sdk/interfaces/SyncFunctionsResult.md";
  slug: "api/sdk/interfaces/syncfunctionsresult";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SyncMigrationsOptions.md": {
	id: "api/sdk/interfaces/SyncMigrationsOptions.md";
  slug: "api/sdk/interfaces/syncmigrationsoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SyncMigrationsResult.md": {
	id: "api/sdk/interfaces/SyncMigrationsResult.md";
  slug: "api/sdk/interfaces/syncmigrationsresult";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/SystemSetting.md": {
	id: "api/sdk/interfaces/SystemSetting.md";
  slug: "api/sdk/interfaces/systemsetting";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Table.md": {
	id: "api/sdk/interfaces/Table.md";
  slug: "api/sdk/interfaces/table";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TestEmailTemplateRequest.md": {
	id: "api/sdk/interfaces/TestEmailTemplateRequest.md";
  slug: "api/sdk/interfaces/testemailtemplaterequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TestWebhookResponse.md": {
	id: "api/sdk/interfaces/TestWebhookResponse.md";
  slug: "api/sdk/interfaces/testwebhookresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TwoFactorEnableResponse.md": {
	id: "api/sdk/interfaces/TwoFactorEnableResponse.md";
  slug: "api/sdk/interfaces/twofactorenableresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TwoFactorSetupResponse.md": {
	id: "api/sdk/interfaces/TwoFactorSetupResponse.md";
  slug: "api/sdk/interfaces/twofactorsetupresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TwoFactorStatusResponse.md": {
	id: "api/sdk/interfaces/TwoFactorStatusResponse.md";
  slug: "api/sdk/interfaces/twofactorstatusresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/TwoFactorVerifyRequest.md": {
	id: "api/sdk/interfaces/TwoFactorVerifyRequest.md";
  slug: "api/sdk/interfaces/twofactorverifyrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateAPIKeyRequest.md": {
	id: "api/sdk/interfaces/UpdateAPIKeyRequest.md";
  slug: "api/sdk/interfaces/updateapikeyrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateAppSettingsRequest.md": {
	id: "api/sdk/interfaces/UpdateAppSettingsRequest.md";
  slug: "api/sdk/interfaces/updateappsettingsrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateAuthSettingsRequest.md": {
	id: "api/sdk/interfaces/UpdateAuthSettingsRequest.md";
  slug: "api/sdk/interfaces/updateauthsettingsrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateAuthSettingsResponse.md": {
	id: "api/sdk/interfaces/UpdateAuthSettingsResponse.md";
  slug: "api/sdk/interfaces/updateauthsettingsresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateEmailTemplateRequest.md": {
	id: "api/sdk/interfaces/UpdateEmailTemplateRequest.md";
  slug: "api/sdk/interfaces/updateemailtemplaterequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateFunctionRequest.md": {
	id: "api/sdk/interfaces/UpdateFunctionRequest.md";
  slug: "api/sdk/interfaces/updatefunctionrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateMigrationRequest.md": {
	id: "api/sdk/interfaces/UpdateMigrationRequest.md";
  slug: "api/sdk/interfaces/updatemigrationrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateOAuthProviderRequest.md": {
	id: "api/sdk/interfaces/UpdateOAuthProviderRequest.md";
  slug: "api/sdk/interfaces/updateoauthproviderrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateOAuthProviderResponse.md": {
	id: "api/sdk/interfaces/UpdateOAuthProviderResponse.md";
  slug: "api/sdk/interfaces/updateoauthproviderresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateSystemSettingRequest.md": {
	id: "api/sdk/interfaces/UpdateSystemSettingRequest.md";
  slug: "api/sdk/interfaces/updatesystemsettingrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateUserAttributes.md": {
	id: "api/sdk/interfaces/UpdateUserAttributes.md";
  slug: "api/sdk/interfaces/updateuserattributes";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateUserRoleRequest.md": {
	id: "api/sdk/interfaces/UpdateUserRoleRequest.md";
  slug: "api/sdk/interfaces/updateuserrolerequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpdateWebhookRequest.md": {
	id: "api/sdk/interfaces/UpdateWebhookRequest.md";
  slug: "api/sdk/interfaces/updatewebhookrequest";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UploadOptions.md": {
	id: "api/sdk/interfaces/UploadOptions.md";
  slug: "api/sdk/interfaces/uploadoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UploadProgress.md": {
	id: "api/sdk/interfaces/UploadProgress.md";
  slug: "api/sdk/interfaces/uploadprogress";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/UpsertOptions.md": {
	id: "api/sdk/interfaces/UpsertOptions.md";
  slug: "api/sdk/interfaces/upsertoptions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/User.md": {
	id: "api/sdk/interfaces/User.md";
  slug: "api/sdk/interfaces/user";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/ValidateInvitationResponse.md": {
	id: "api/sdk/interfaces/ValidateInvitationResponse.md";
  slug: "api/sdk/interfaces/validateinvitationresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/WeakPassword.md": {
	id: "api/sdk/interfaces/WeakPassword.md";
  slug: "api/sdk/interfaces/weakpassword";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/Webhook.md": {
	id: "api/sdk/interfaces/Webhook.md";
  slug: "api/sdk/interfaces/webhook";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/interfaces/WebhookDelivery.md": {
	id: "api/sdk/interfaces/WebhookDelivery.md";
  slug: "api/sdk/interfaces/webhookdelivery";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/AuthResponseData.md": {
	id: "api/sdk/type-aliases/AuthResponseData.md";
  slug: "api/sdk/type-aliases/authresponsedata";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/BroadcastCallback.md": {
	id: "api/sdk/type-aliases/BroadcastCallback.md";
  slug: "api/sdk/type-aliases/broadcastcallback";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/DataResponse.md": {
	id: "api/sdk/type-aliases/DataResponse.md";
  slug: "api/sdk/type-aliases/dataresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/EmailTemplateType.md": {
	id: "api/sdk/type-aliases/EmailTemplateType.md";
  slug: "api/sdk/type-aliases/emailtemplatetype";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/FilterOperator.md": {
	id: "api/sdk/type-aliases/FilterOperator.md";
  slug: "api/sdk/type-aliases/filteroperator";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/FluxbaseAuthResponse.md": {
	id: "api/sdk/type-aliases/FluxbaseAuthResponse.md";
  slug: "api/sdk/type-aliases/fluxbaseauthresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/FluxbaseResponse.md": {
	id: "api/sdk/type-aliases/FluxbaseResponse.md";
  slug: "api/sdk/type-aliases/fluxbaseresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/HttpMethod.md": {
	id: "api/sdk/type-aliases/HttpMethod.md";
  slug: "api/sdk/type-aliases/httpmethod";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/ImpersonationType.md": {
	id: "api/sdk/type-aliases/ImpersonationType.md";
  slug: "api/sdk/type-aliases/impersonationtype";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/OrderDirection.md": {
	id: "api/sdk/type-aliases/OrderDirection.md";
  slug: "api/sdk/type-aliases/orderdirection";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/PresenceCallback.md": {
	id: "api/sdk/type-aliases/PresenceCallback.md";
  slug: "api/sdk/type-aliases/presencecallback";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/RealtimeCallback.md": {
	id: "api/sdk/type-aliases/RealtimeCallback.md";
  slug: "api/sdk/type-aliases/realtimecallback";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/SessionResponse.md": {
	id: "api/sdk/type-aliases/SessionResponse.md";
  slug: "api/sdk/type-aliases/sessionresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/StorageObject.md": {
	id: "api/sdk/type-aliases/StorageObject.md";
  slug: "api/sdk/type-aliases/storageobject";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/SupabaseAuthResponse.md": {
	id: "api/sdk/type-aliases/SupabaseAuthResponse.md";
  slug: "api/sdk/type-aliases/supabaseauthresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/SupabaseResponse.md": {
	id: "api/sdk/type-aliases/SupabaseResponse.md";
  slug: "api/sdk/type-aliases/supabaseresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/UserResponse.md": {
	id: "api/sdk/type-aliases/UserResponse.md";
  slug: "api/sdk/type-aliases/userresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"api/sdk/type-aliases/VoidResponse.md": {
	id: "api/sdk/type-aliases/VoidResponse.md";
  slug: "api/sdk/type-aliases/voidresponse";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"deployment/docker.md": {
	id: "deployment/docker.md";
  slug: "deployment/docker";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"deployment/kubernetes.md": {
	id: "deployment/kubernetes.md";
  slug: "deployment/kubernetes";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"deployment/overview.md": {
	id: "deployment/overview.md";
  slug: "deployment/overview";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"deployment/production-checklist.md": {
	id: "deployment/production-checklist.md";
  slug: "deployment/production-checklist";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"deployment/scaling.md": {
	id: "deployment/scaling.md";
  slug: "deployment/scaling";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"getting-started/installation.md": {
	id: "getting-started/installation.md";
  slug: "getting-started/installation";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"getting-started/quick-start.md": {
	id: "getting-started/quick-start.md";
  slug: "getting-started/quick-start";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/admin/configuration-management.md": {
	id: "guides/admin/configuration-management.md";
  slug: "guides/admin/configuration-management";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/admin/index.md": {
	id: "guides/admin/index.md";
  slug: "guides/admin";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/admin/user-impersonation.md": {
	id: "guides/admin/user-impersonation.md";
  slug: "guides/admin/user-impersonation";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/authentication.md": {
	id: "guides/authentication.md";
  slug: "guides/authentication";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/database-migrations.md": {
	id: "guides/database-migrations.md";
  slug: "guides/database-migrations";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/edge-functions.md": {
	id: "guides/edge-functions.md";
  slug: "guides/edge-functions";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/email-services.md": {
	id: "guides/email-services.md";
  slug: "guides/email-services";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/jobs.md": {
	id: "guides/jobs.md";
  slug: "guides/jobs";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/logging.md": {
	id: "guides/logging.md";
  slug: "guides/logging";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/monitoring-observability.md": {
	id: "guides/monitoring-observability.md";
  slug: "guides/monitoring-observability";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/oauth-providers.md": {
	id: "guides/oauth-providers.md";
  slug: "guides/oauth-providers";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/rate-limiting.md": {
	id: "guides/rate-limiting.md";
  slug: "guides/rate-limiting";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/realtime.md": {
	id: "guides/realtime.md";
  slug: "guides/realtime";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/row-level-security.md": {
	id: "guides/row-level-security.md";
  slug: "guides/row-level-security";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/storage.md": {
	id: "guides/storage.md";
  slug: "guides/storage";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/testing.md": {
	id: "guides/testing.md";
  slug: "guides/testing";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/typescript-sdk/database.md": {
	id: "guides/typescript-sdk/database.md";
  slug: "guides/typescript-sdk/database";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/typescript-sdk/getting-started.md": {
	id: "guides/typescript-sdk/getting-started.md";
  slug: "guides/typescript-sdk/getting-started";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/typescript-sdk/index.md": {
	id: "guides/typescript-sdk/index.md";
  slug: "/category/sdks";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/typescript-sdk/react-hooks.md": {
	id: "guides/typescript-sdk/react-hooks.md";
  slug: "guides/typescript-sdk/react-hooks";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"guides/webhooks.md": {
	id: "guides/webhooks.md";
  slug: "guides/webhooks";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"intro.md": {
	id: "intro.md";
  slug: "intro";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"reference/configuration.md": {
	id: "reference/configuration.md";
  slug: "reference/configuration";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/admin.md": {
	id: "sdk/admin.md";
  slug: "sdk/admin";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/advanced-features.md": {
	id: "sdk/advanced-features.md";
  slug: "sdk/advanced-features";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/ddl.md": {
	id: "sdk/ddl.md";
  slug: "sdk/ddl";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/impersonation.md": {
	id: "sdk/impersonation.md";
  slug: "sdk/impersonation";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/management.md": {
	id: "sdk/management.md";
  slug: "sdk/management";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/oauth.md": {
	id: "sdk/oauth.md";
  slug: "sdk/oauth";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"sdk/settings.md": {
	id: "sdk/settings.md";
  slug: "sdk/settings";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"security/best-practices.md": {
	id: "security/best-practices.md";
  slug: "security/best-practices";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"security/csrf-protection.md": {
	id: "security/csrf-protection.md";
  slug: "security/csrf-protection";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"security/overview.md": {
	id: "security/overview.md";
  slug: "security/overview";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"security/security-headers.md": {
	id: "security/security-headers.md";
  slug: "security/security-headers";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"settings/app-settings-guide.md": {
	id: "settings/app-settings-guide.md";
  slug: "settings/app-settings-guide";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
"supabase-comparison.md": {
	id: "supabase-comparison.md";
  slug: "supabase-comparison";
  body: string;
  collection: "docs";
  data: InferEntrySchema<"docs">
} & { render(): Render[".md"] };
};

	};

	type DataEntryMap = {
		
	};

	type AnyEntryMap = ContentEntryMap & DataEntryMap;

	export type ContentConfig = typeof import("../../src/content/config.js");
}
