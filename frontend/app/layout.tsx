import type { Metadata } from "next";
import { Barlow } from "next/font/google";
import "./globals.css";
import Link from "next/link";
import { Separator } from "@/components/ui/separator";
import { cn } from "../lib/utils";

const barlow = Barlow({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-barlow",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Media Pipeline — Upload & Process",
  description:
    "Upload images and videos for automated cloud processing. Track status live and browse your processed gallery.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={`dark ${barlow.variable}`}>
      <body className={cn('min-h-screen', 'text-foreground')}>
        {/* ── Navigation ── */}
        <header className={cn('sticky', 'top-0', 'z-50', 'border-border/60', 'backdrop-blur-md')}>
          <div className={cn('mx-auto', 'flex', 'h-14', 'max-w-6xl', 'items-center', 'justify-between', 'px-4', 'sm:px-6')}>
            {/* Logo */}
            <Link
              href="/"
              className={cn('flex', 'items-center', 'gap-2.5', 'font-semibold', 'tracking-tight', 'text-foreground', 'transition-opacity', 'hover:opacity-80')}
            >
              {/* Cloud + spark icon */}
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={cn('h-5', 'w-5', 'text-primary')}
                aria-hidden="true"
              >
                <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z" />
                <path d="m9.5 11 1.5 1.5L13 10" />
              </svg>
              <span>MediaPipeline</span>
            </Link>

            {/* Nav links */}
            <nav className={cn('flex', 'items-center', 'gap-1')} aria-label="Primary navigation">
              <Link
                href="/"
                id="nav-upload"
                className={cn('rounded-md', 'px-3', 'py-1.5', 'text-sm', 'font-medium', 'text-muted-foreground', 'transition-colors', 'hover:bg-muted', 'hover:text-foreground')}
              >
                Upload
              </Link>
              <Link
                href="/gallery"
                id="nav-gallery"
                className={cn('rounded-md', 'px-3', 'py-1.5', 'text-sm', 'font-medium', 'text-muted-foreground', 'transition-colors', 'hover:bg-muted', 'hover:text-foreground')}
              >
                Gallery
              </Link>
            </nav>
          </div>
        </header>

        {/* ── Page content ── */}
        <main className={cn('mx-auto', 'max-w-6xl', 'px-4', 'py-8', 'sm:px-6')}>
          {children}
        </main>

        {/* ── Footer ── */}
        <footer className={cn('mt-16', 'border-t', 'border-border/40', 'py-6', 'text-center', 'text-xs', 'text-muted-foreground')}>
          <Separator className={cn('mb-6', 'opacity-30')} />
          MediaPipeline · AWS S3 · Lambda · CloudFront
        </footer>
      </body>
    </html>
  );
}
