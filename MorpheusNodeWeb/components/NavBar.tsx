import React from 'react';
import Link from 'next/link';

export const NavBar: React.FC = () => {
  return (
    <nav style={{ background: '#f1f1f1', padding: '1rem' }}>
      <Link href="/">Home</Link>
      {/* Add more links as needed */}
    </nav>
  );
};
