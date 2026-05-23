"use client";

import * as React from "react";
import Link from "next/link";
import { Github, Twitter, Linkedin, Shield } from "lucide-react";

const footerLinks = {
  Product: [
    { href: "/#features", label: "Features" },
    { href: "/pricing", label: "Pricing" },
    { href: "#", label: "Changelog" },
    { href: "#", label: "Roadmap" },
  ],
  Resources: [
    { href: "#", label: "Documentation" },
    { href: "#", label: "API Reference" },
    { href: "/blog", label: "Blog" },
    { href: "#", label: "Community" },
  ],
  Company: [
    { href: "/about", label: "About" },
    { href: "/contact", label: "Contact" },
    { href: "#", label: "Careers" },
    { href: "#", label: "Press Kit" },
  ],
  Legal: [
    { href: "#", label: "Privacy Policy" },
    { href: "#", label: "Terms of Service" },
    { href: "#", label: "Cookie Policy" },
    { href: "#", label: "Security" },
  ],
};

const socialLinks = [
  { href: "https://github.com", icon: Github, label: "GitHub" },
  { href: "https://twitter.com", icon: Twitter, label: "Twitter" },
  { href: "https://linkedin.com", icon: Linkedin, label: "LinkedIn" },
];

export function Footer() {
  return (
    <footer className="border-t border-fortress-navy-light bg-fortress-navy">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-5">
          <div className="col-span-2 md:col-span-1">
            <Link href="/" className="flex items-center gap-2">
              <Shield className="h-8 w-8 text-fortress-green" />
              <span className="text-xl font-bold text-white">FortressWAF</span>
            </Link>
            <p className="mt-4 text-sm text-fortress-gray-light">
              Open-core security for the modern web.
            </p>
            <div className="mt-4 flex gap-4">
              {socialLinks.map((social) => (
                <a
                  key={social.label}
                  href={social.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-fortress-gray hover:text-white transition-colors"
                  aria-label={social.label}
                >
                  <social.icon className="h-5 w-5" />
                </a>
              ))}
            </div>
          </div>

          {Object.entries(footerLinks).map(([title, links]) => (
            <div key={title}>
              <h3 className="text-sm font-semibold text-white">{title}</h3>
              <ul className="mt-4 space-y-2">
                {links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="text-sm text-fortress-gray-light hover:text-white transition-colors"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 border-t border-fortress-navy-light pt-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-fortress-gray">
              © 2025 FortressWAF. All rights reserved.
            </p>
            <p className="text-sm text-fortress-gray">
              Open Core — Free forever for individual developers.
            </p>
          </div>
        </div>
      </div>
    </footer>
  );
}
