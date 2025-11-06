import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Auto-Generated REST API',
    description: (
      <>
        Automatically generates CRUD endpoints from your PostgreSQL schema.
        PostgREST-compatible with full filtering, sorting, and pagination support.
      </>
    ),
  },
  {
    title: 'Authentication Built-in',
    description: (
      <>
        Email/password, magic links, OAuth, and JWT tokens out of the box.
        Session management and Row Level Security (RLS) support included.
      </>
    ),
  },
  {
    title: 'Realtime Subscriptions',
    description: (
      <>
        WebSocket-based live data updates using PostgreSQL LISTEN/NOTIFY.
        Subscribe to database changes in real-time with a simple API.
      </>
    ),
  },
  {
    title: 'File Storage',
    description: (
      <>
        Upload and download files with access policies. Supports both local
        filesystem and S3-compatible storage backends.
      </>
    ),
  },
  {
    title: 'Edge Functions',
    description: (
      <>
        Execute JavaScript/TypeScript functions with Deno runtime.
        Perfect for custom business logic and API extensions.
      </>
    ),
  },
  {
    title: 'Single Binary',
    description: (
      <>
        Deploy a complete backend in a single ~80MB binary. Only PostgreSQL
        required as an external dependency.
      </>
    ),
  },
];

function Feature({title, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
