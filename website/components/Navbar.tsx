"use client";

import * as React from "react";
import Link from "next/link";
import { Menu, X, Shield } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";

const navLinks = [
  { href: "/#features", label: "Product" },
  { href: "/pricing", label: "Pricing" },
  { href: "#", label: "Docs" },
  { href: "/blog", label: "Blog" },
];

export function Navbar() {
  const [isOpen, setIsOpen] = React.useState(false);

  return (
    <nav className="fixed top-0 left-0 right-0 z-50 border-b border-fortress-navy-light bg-fortress-navy/95 backdrop-blur-sm">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center gap-8">
            <Link href="/" className="flex items-center gap-2">
              <Shield className="h-8 w-8 text-fortress-green" />
              <span className="text-xl font-bold text-white">FortressWAF</span>
            </Link>

            <div className="hidden md:flex md:items-center md:gap-6">
              {navLinks.map((link) => (
                <Link
                  key={link.label}
                  href={link.href}
                  className="text-sm font-medium text-fortress-gray-light transition-colors hover:text-white"
                >
                  {link.label}
                </Link>
              ))}
              <a
                href="https://github.com"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm font-medium text-fortress-gray-light transition-colors hover:text-white"
              >
                GitHub
              </a>
            </div>
          </div>

          <div className="hidden md:flex md:items-center md:gap-4">
            <Button variant="default" size="sm" asChild>
              <Link href="/contact">Start Free</Link>
            </Button>
          </div>

          <div className="flex items-center md:hidden">
            <Sheet open={isOpen} onOpenChange={setIsOpen}>
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon" className="text-white">
                  <Menu className="h-6 w-6" />
                </Button>
              </SheetTrigger>
              <SheetContent side="right" className="bg-fortress-navy border-fortress-navy-light">
                <div className="flex flex-col gap-4 mt-8">
                  {navLinks.map((link) => (
                    <Link
                      key={link.label}
                      href={link.href}
                      className="text-lg font-medium text-fortress-gray-light transition-colors hover:text-white"
                      onClick={() => setIsOpen(false)}
                    >
                      {link.label}
                    </Link>
                  ))}
                  <a
                    href="https://github.com"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-lg font-medium text-fortress-gray-light transition-colors hover:text-white"
                    onClick={() => setIsOpen(false)}
                  >
                    GitHub
                  </a>
                  <Button className="mt-4" asChild>
                    <Link href="/contact" onClick={() => setIsOpen(false)}>
                      Start Free
                    </Link>
                  </Button>
                </div>
              </SheetContent>
            </Sheet>
          </div>
        </div>
      </div>
    </nav>
  );
}
