rootProject.name = "fluxbase-kotlin"

// Enable type-safe project accessors (not strictly needed for a single-module project
// but keeps the door open for splitting into submodules later — auth, postgrest, etc.).
enableFeaturePreview("TYPESAFE_PROJECT_ACCESSORS")

pluginManagement {
    repositories {
        gradlePluginPortal()
        mavenCentral()
    }
}

dependencyResolutionManagement {
    repositories {
        mavenCentral()
    }
}
