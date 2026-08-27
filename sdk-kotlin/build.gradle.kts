/*
 * fluxbase-kotlin — a full-parity Kotlin port of @nimbleflux/fluxbase-sdk.
 *
 * Phase: S0/S1 (core + auth). Modules will be split into Gradle submodules later
 * (postgrest, realtime, storage, functions, jobs, rpc, secrets, settings, ai, …)
 * as each ships; for now everything lives in this single module under package
 * io.github.nimbleflux.fluxbase for the fastest TDD loop.
 */

plugins {
    `maven-publish`
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.kover)
    alias(libs.plugins.detekt)
    alias(libs.plugins.dokka)
    alias(libs.plugins.mavenPublish)
}

group = "io.github.nimbleflux"

// Untagged/local builds use "dev"; CI passes -Pversion=<tag-stripped> on publish.
// See sdk-kotlin/README.md §Releasing.
version = project.findProperty("version") as String? ?: "dev"

repositories {
    mavenCentral()
}

dependencies {
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kotlinx.datetime)

    // Ktor client (engine + WebSocket support for realtime).
    implementation(libs.ktor.client.core)
    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.websockets)
    implementation(libs.ktor.client.contentNegotiation)
    implementation(libs.ktor.serialization.kotlinx.json)

    // Test
    testImplementation(libs.kotlin.test)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.ktor.client.mock)
    testRuntimeOnly(libs.kotlin.test.junit5)
}

// ---- Kotlin / JVM ----
kotlin {
    jvmToolchain(21)
    // explicitApi() is enabled once the public surface stabilizes post-S1.
}

// ---- Testing (JUnit 5) ----
tasks.withType<Test> {
    useJUnitPlatform()
}

// ---- Integration test source set (against a live Fluxbase on localhost:8080) ----
// Mirrors the two-tier test strategy: src/test (pure fakes, fast) +
// src/integrationTest (real server, contract verification).
sourceSets {
    create("integrationTest") {
        compileClasspath += sourceSets.main.get().output + configurations.testRuntimeClasspath.get()
        runtimeClasspath += sourceSets.main.get().output + configurations.testRuntimeClasspath.get()
        kotlin {
            srcDir("src/integrationTest/kotlin")
        }
    }
}

val integrationTest by tasks.registering(Test::class) {
    description = "Runs integration tests against a live Fluxbase instance (localhost:8080)."
    group = "verification"
    testClassesDirs = sourceSets["integrationTest"].output.classesDirs
    classpath = sourceSets["integrationTest"].runtimeClasspath
    shouldRunAfter("test")
    useJUnitPlatform()
}

tasks.named("check") { dependsOn(integrationTest) }

// ---- Kover coverage (target ≥80% on core/auth/postgrest/realtime) ----
// The verify gate is enabled once S1 ships real code; for the S0 spike we just
// generate reports. See sdk-kotlin/README.md §Testing.
kover {
    reports {
        total {
            xml { onCheck = true }
            html { onCheck = true }
        }
    }
}

// ---- detekt ----
detekt {
    buildUponDefaultConfig = true
    config.setFrom(files("$rootDir/detekt.yml"))
}

// ---- Dokka API documentation ----
// Mirrors how the TS SDK docs are integrated: Dokka emits GitHub-Flavored Markdown
// into the Starlight docs content tree (docs/src/content/docs/api/sdk-kotlin/),
// which Astro's content collection + sidebar `autogenerate` picks up automatically.
// The generated output is committed to git, matching the 395 committed TS API docs.
//
// Regenerate with:  ./gradlew dokkaGfm
// Then commit the changed files under docs/src/content/docs/api/sdk-kotlin/.
tasks.withType<org.jetbrains.dokka.gradle.DokkaTask>().configureEach {
    // In Dokka 1.9.x, dokkaHtml / dokkaGfm / dokkaJavadoc are all DokkaTask instances.
}

/**
 * Slug a Dokka path segment the way Starlight slugifies content-collection
 * routes: lowercase, camelCase boundaries become dashes, dots are dropped,
 * [a-z0-9_-] survives. e.g. "io.github.nimbleflux.fluxbase.auth" →
 * "iogithubnimblefluxfluxbaseauth", "PostgresChangesConfig" →
 * "-postgres-changes-config", "-i-n-n-e-r_PRODUCT" keeps its underscores.
 */
fun slugSegment(segment: String): String {
    val sb = StringBuilder()
    segment.forEachIndexed { i, ch ->
        when {
            ch.isUpperCase() -> {
                val prevIsUpper = i > 0 && segment[i - 1].isUpperCase()
                val nextIsLower = i + 1 < segment.length && segment[i + 1].isLowerCase()
                if (sb.isNotEmpty() && (!prevIsUpper || nextIsLower)) sb.append('-')
                sb.append(ch.lowercaseChar())
            }
            ch == '.' || (!ch.isLetterOrDigit() && ch != '_' && ch != '-') -> { /* dropped */ }
            else -> sb.append(ch.lowercaseChar())
        }
    }
    var s = sb.toString().trim('-')
    if (s.isNotEmpty() && segment.startsWith("-")) s = "-$s"
    return s
}

/** Normalize a POSIX-ish relative path (resolve "." and ".." segments). */
fun normalizeRelative(path: String): String {
    val out = ArrayDeque<String>()
    for (part in path.split('/')) {
        when (part) {
            "", "." -> {}
            ".." -> if (out.isNotEmpty() && out.last() != "..") out.removeLast() else out.addLast("..")
            else -> out.addLast(part)
        }
    }
    return if (out.isEmpty()) "." else out.joinToString("/")
}

tasks.named<org.jetbrains.dokka.gradle.DokkaTask>("dokkaGfm").configure {
    outputDirectory.set(file("../docs/src/content/docs/api/sdk-kotlin"))
    // Post-process: inject Starlight frontmatter into every generated .md file,
    // then rewrite Dokka's GitHub-flavored .md links to Starlight directory URLs.
    //
    // Frontmatter: Dokka GFM output has none, but Starlight's content schema
    // requires a `title` field. This extracts the first H1 heading as the title
    // and prepends the block — matching what starlight-typedoc does for the TS
    // SDK docs.
    //
    // Links: Dokka emits [text](path.md) links that work on GitHub but are dead
    // in the built site — Astro keeps them verbatim, so the docs link checker
    // (and users) 404 on them. Every page renders to <route>/index.html, so
    // each link becomes a relative path between the two route directories
    // (trailing slash, slugified segments).
    doLast {
        val outputDir = file("../docs/src/content/docs/api/sdk-kotlin")
        var processed = 0
        outputDir.walkTopDown().filter { it.isFile && it.extension == "md" }.forEach { mdFile ->
            val content = mdFile.readText()
            // Skip if already has frontmatter (idempotent).
            if (content.startsWith("---")) return@forEach

            // Extract title from first H1 heading; fall back to filename.
            val titleMatch = Regex("^#\\s+(.+)$", RegexOption.MULTILINE).find(content)
            val title = titleMatch?.groupValues?.getOrNull(1)
                ?.replace("`", "")
                ?.replace("[\\[\\]]".toRegex(), "")
                ?.ifBlank { mdFile.nameWithoutExtension }
                ?: mdFile.nameWithoutExtension

            val frontmatter = buildString {
                appendLine("---")
                appendLine("title: \"${title.replace("\"", "\\\"")}\"")
                appendLine("editUrl: false")
                appendLine("next: false")
                appendLine("prev: false")
                appendLine("---")
                appendLine()
            }
            mdFile.writeText(frontmatter + content)
            processed++
        }
        logger.lifecycle("Injected Starlight frontmatter into $processed Dokka files.")

        // Built route directory of a source .md file, or null when [relPath]
        // is not one of the generated files.
        val mdFiles = outputDir.walkTopDown().filter { it.isFile && it.extension == "md" }.toList()
        val relSet = mdFiles.map { it.relativeTo(outputDir).invariantSeparatorsPath }.toSet()
        fun routeDirFor(relPath: String): String? {
            if (relPath !in relSet) return null
            val segs = relPath.removeSuffix(".md").split('/')
                .map { slugSegment(it) }
                .filter { it.isNotEmpty() }
                .toMutableList()
            if (segs.isNotEmpty() && segs.last() == "index") segs.removeAt(segs.size - 1)
            return if (segs.isEmpty()) "." else segs.joinToString("/")
        }

        val linkRegex = Regex("(\\]\\()([^)#\\s]+?\\.md)(#[^)\\s]*)?(\\))")
        var filesChanged = 0
        var linksRewritten = 0
        var linksUnresolved = 0
        mdFiles.forEach { mdFile ->
            val rel = mdFile.relativeTo(outputDir).invariantSeparatorsPath
            val fromDir = routeDirFor(rel) ?: return@forEach
            val text = mdFile.readText()
            val matches = linkRegex.findAll(text).toList()
            if (matches.isEmpty()) return@forEach

            val sb = StringBuilder()
            var last = 0
            var fileChanged = false
            for (m in matches) {
                val link = m.groupValues[2]
                val anchor = m.groupValues[3]
                val dir = rel.substringBeforeLast('/', "")
                val targetRel = normalizeRelative(if (dir.isEmpty()) link else "$dir/$link")
                val toDir = routeDirFor(targetRel)
                sb.append(text, last, m.range.first)
                if (toDir != null) {
                    // Relative path from one route directory to another.
                    val fromSegs = if (fromDir == ".") emptyList() else fromDir.split('/')
                    val toSegs = if (toDir == ".") emptyList() else toDir.split('/')
                    var common = 0
                    while (common < fromSegs.size && common < toSegs.size && fromSegs[common] == toSegs[common]) common++
                    val parts = buildList {
                        repeat(fromSegs.size - common) { add("..") }
                        addAll(toSegs.subList(common, toSegs.size))
                    }
                    val url = if (parts.isEmpty()) "./" else parts.joinToString("/") + "/"
                    sb.append("](").append(url).append(anchor).append(")")
                    linksRewritten++
                    fileChanged = true
                } else {
                    linksUnresolved++
                    sb.append(m.value)
                }
                last = m.range.last + 1
            }
            sb.append(text, last, text.length)
            if (fileChanged) {
                mdFile.writeText(sb.toString())
                filesChanged++
            }
        }
        logger.lifecycle("Rewrote $linksRewritten Dokka .md links in $filesChanged files ($linksUnresolved unresolved).")
        if (linksUnresolved > 0) {
            logger.warn("Dokka docs: $linksUnresolved .md links could not be resolved to generated pages — check for broken cross-references.")
        }
    }
}

// ---- Maven publishing (GitHub Packages) ----
// Uses com.vanniktech.maven.publish to handle sources + javadoc JARs, POM metadata,
// and upload. GitHub Packages needs no GPG signing.
//
// Versioning: there is no separate Kotlin tag. The artifact is published as part of
// the main Fluxbase release — release.yml's publish-kotlin-sdk job passes
// -Pversion=<main-tag-version>; untagged/local builds are "dev".
// See CONTRIBUTING.md §Publishing.
mavenPublishing {
    coordinates(
        groupId = "io.github.nimbleflux",
        artifactId = "fluxbase-kotlin",
        version = project.version.toString(),
    )
    pom {
        name.set("fluxbase-kotlin")
        description.set("A full-parity Kotlin port of @nimbleflux/fluxbase-sdk for Fluxbase, a self-hostable Supabase-compatible BaaS.")
        inceptionYear.set("2026")
        url.set("https://github.com/nimbleflux/fluxbase")
        licenses {
            license {
                name.set("Apache License, Version 2.0")
                url.set("https://www.apache.org/licenses/LICENSE-2.0.txt")
            }
        }
        developers {
            developer {
                id.set("nimbleflux")
                name.set("Nimbleflux")
            }
        }
        scm {
            url.set("https://github.com/nimbleflux/fluxbase")
            connection.set("scm:git:git://github.com/nimbleflux/fluxbase.git")
            developerConnection.set("scm:git:ssh://github.com/nimbleflux/fluxbase.git")
        }
    }
}

// GitHub Packages repository — the publish step targets this repo.
// CI publishes via GITHUB_ACTOR + GITHUB_TOKEN (auto-injected).
// Local publishing: set gpr.user + gpr.key in ~/.gradle/gradle.properties.
publishing {
    repositories {
        maven {
            name = "gpr"
            url = uri("https://maven.pkg.github.com/nimbleflux/fluxbase")
            credentials {
                username = System.getenv("GITHUB_ACTOR") ?: providers.gradleProperty("gpr.user").orNull
                password = System.getenv("GITHUB_TOKEN") ?: providers.gradleProperty("gpr.key").orNull
            }
        }
    }
}
