import type { Metadata } from 'next';
import '../globals.css';

export const metadata: Metadata = {
  title: 'FortressWAF Setup',
  description: 'FortressWAF Setup Wizard',
};

export default function SetupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
      </body>
    </html>
  );
}
