import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <h1 className="hero__title">{siteConfig.title}</h1>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div style={{
          backgroundColor: 'rgba(255, 193, 7, 0.1)',
          border: '2px solid #ffc107',
          borderRadius: '8px',
          padding: '12px 20px',
          marginTop: '20px',
          marginBottom: '20px',
        }}>
          <p style={{ margin: 0, fontSize: '0.95rem', lineHeight: '1.5' }}>
            <strong>⚠️ Pre-Release Software:</strong> Fluxbase is in active development.
            Expect breaking changes between versions. Use for testing and development purposes.
          </p>
        </div>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro">
            Get Started - 5min ⏱️
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home(): JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} Documentation`}
      description="Lightweight Backend-as-a-Service Alternative to Supabase">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
