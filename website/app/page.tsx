"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { motion, useInView } from "framer-motion";
import { useRef } from "react";
import {
  Shield,
  Zap,
  Lock,
  Code,
  Globe,
  BarChart3,
  Bot,
  Server,
  Check,
  X,
  ChevronRight,
  Github,
  Star,
  ArrowRight,
  Layers,
  ShieldCheck,
  Activity,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const features = [
  {
    icon: Shield,
    title: "AI-Powered Detection",
    description: "Machine learning models detect zero-day attacks in real-time.",
  },
  {
    icon: Code,
    title: "API Security",
    description: "Full protection for REST, GraphQL, and gRPC APIs with schema validation.",
  },
  {
    icon: Bot,
    title: "Bot Management",
    description: "Advanced fingerprinting stops scrapers, bots, and automated attacks.",
  },
  {
    icon: Server,
    title: "Virtual Patching",
    description: "Patch vulnerabilities without code changes while you remediate.",
  },
  {
    icon: Layers,
    title: "Multi-Tenant",
    description: "Manage thousands of sites from a single pane of glass.",
  },
  {
    icon: ShieldCheck,
    title: "Compliance Ready",
    description: "PCI DSS, SOC 2, HIPAA, and GDPR compliance reporting built-in.",
  },
  {
    icon: Globe,
    title: "Deploy Anywhere",
    description: "Kubernetes, Docker, bare metal, or major cloud providers.",
  },
  {
    icon: BarChart3,
    title: "Real-Time Dashboard",
    description: "Live traffic visualization with threat intelligence feeds.",
  },
  {
    icon: Zap,
    title: "Developer-First",
    description: "CLI tools, Terraform provider, and native integrations for DevOps.",
  },
];

const comparisonData = [
  {
    feature: "OWASP Top 10",
    fortress: true,
    cloudflare: true,
    imperva: true,
    appTrana: true,
  },
  {
    feature: "API Security (full)",
    fortress: true,
    cloudflare: false,
    imperva: true,
    appTrana: false,
  },
  {
    feature: "Self-hosted option",
    fortress: true,
    cloudflare: false,
    imperva: false,
    appTrana: false,
  },
  {
    feature: "ML zero-day engine",
    fortress: true,
    cloudflare: false,
    imperva: true,
    appTrana: false,
  },
  {
    feature: "Open-source core",
    fortress: true,
    cloudflare: false,
    imperva: false,
    appTrana: false,
  },
  {
    feature: "White-label / MSSP",
    fortress: true,
    cloudflare: false,
    imperva: true,
    appTrana: true,
  },
  {
    feature: "Starting price",
    fortress: "Free",
    cloudflare: "$20/mo",
    imperva: "$300/mo",
    appTrana: "$200/mo",
  },
  {
    feature: "Setup time",
    fortress: "< 30 min",
    cloudflare: "Hours",
    imperva: "Days",
    appTrana: "Days",
  },
];

const pricingTiers = [
  {
    name: "Community",
    price: 0,
    description: "For individual developers and small projects.",
    features: [
      "3 rulesets",
      "10K requests/month",
      "Community support",
      "Basic analytics",
      "Self-hosted",
    ],
    cta: "Get Started",
    href: "/contact",
  },
  {
    name: "Starter",
    price: 149,
    description: "For growing applications and startups.",
    features: [
      "Unlimited rulesets",
      "1M requests/month",
      "Email support",
      "Advanced analytics",
      "API access",
      "Basic ML detection",
    ],
    cta: "Start Free Trial",
    href: "/contact",
    popular: false,
  },
  {
    name: "Professional",
    price: 499,
    description: "For production workloads at scale.",
    features: [
      "Everything in Starter",
      "Unlimited requests",
      "Priority support",
      "Advanced ML detection",
      "Multi-tenant management",
      "Custom rules",
      "SOC 2 reports",
    ],
    cta: "Start Free Trial",
    href: "/contact",
    popular: true,
  },
  {
    name: "Enterprise",
    price: null,
    description: "For large organizations with custom needs.",
    features: [
      "Everything in Professional",
      "Dedicated support engineer",
      "Custom SLA",
      "On-premise deployment",
      "White-label options",
      "Advanced integrations",
      "Compliance reports",
    ],
    cta: "Contact Sales",
    href: "/contact",
  },
];

const testimonials = [
  {
    quote:
      "We replaced Cloudflare WAF after years of frustration. Setup took 20 minutes and we blocked our first attack within the hour. The open-source core means no vendor lock-in.",
    author: "Marcus Chen",
    title: "Head of Security",
    company: "TechScale Inc.",
  },
  {
    quote:
      "As an MSSP, we needed a WAF we could white-label and deploy for hundreds of clients. FortressWAF's multi-tenant architecture made this possible at a fraction of Imperva's cost.",
    author: "Sarah Okonkwo",
    title: "VP of Engineering",
    company: "SecureOps MSSP",
  },
  {
    quote:
      "The virtual patching feature saved us during the Log4j crisis. We had coverage within hours while we worked on permanent remediation. Absolutely critical tool.",
    author: "James Rodriguez",
    title: "DevOps Lead",
    company: "FinServ Cloud",
  },
];

const stats = [
  { value: "10M+", label: "Attacks blocked daily" },
  { value: "<2ms", label: "Added latency P99" },
  { value: "52", label: "Security features" },
  { value: "99.9%", label: "Uptime SLA" },
];

function AnimatedCounter({ value, suffix = "" }: { value: string; suffix?: string }) {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true });
  const [displayValue, setDisplayValue] = useState("0");

  useEffect(() => {
    if (isInView) {
      setDisplayValue(value);
    }
  }, [isInView, value]);

  return (
    <span ref={ref} className="tabular-nums">
      {displayValue}
      {suffix}
    </span>
  );
}

function TerminalAnimation() {
  const lines = [
    { text: "[ fortress] INFO: Attack detected", type: "info" },
    { text: "[ fortress] BLOCK: SQL injection attempt", type: "block" },
    { text: "    Source: 192.168.1.105:45321", type: "info" },
    { text: "    Target: /api/users/login", type: "info" },
    { text: "    Payload: ' OR '1'='1' --", type: "error" },
    { text: "[ fortress] SUCCESS: Request blocked", type: "success" },
  ];

  const [visibleLines, setVisibleLines] = useState(0);

  useEffect(() => {
    const timer = setInterval(() => {
      setVisibleLines((prev) => {
        if (prev >= lines.length) {
          clearInterval(timer);
          return prev;
        }
        return prev + 1;
      });
    }, 600);

    return () => clearInterval(timer);
  }, []);

  return (
    <div className="relative rounded-lg border border-fortress-navy-light bg-[#0d1117] overflow-hidden shadow-2xl">
      <div className="flex items-center gap-2 border-b border-fortress-navy-light bg-[#161b22] px-4 py-3">
        <div className="flex gap-2">
          <div className="h-3 w-3 rounded-full bg-[#ff5f56]"></div>
          <div className="h-3 w-3 rounded-full bg-[#ffbd2e]"></div>
          <div className="h-3 w-3 rounded-full bg-[#27c93f]"></div>
        </div>
        <span className="ml-2 text-xs text-fortress-gray">fortress logs</span>
      </div>
      <div className="p-4 font-mono text-sm min-h-[200px]">
        {lines.slice(0, visibleLines).map((line, index) => (
          <div key={index} className="flex items-start gap-2 py-1">
            <span
              className={`${
                line.type === "block"
                  ? "text-amber-400"
                  : line.type === "error"
                  ? "text-red-400"
                  : line.type === "success"
                  ? "text-green-400"
                  : "text-fortress-gray"
              }`}
            >
              {line.text}
            </span>
            {index === visibleLines - 1 && index < lines.length - 1 && (
              <span className="animate-pulse text-white">▋</span>
            )}
          </div>
        ))}
        {visibleLines >= lines.length && (
          <div className="mt-2 flex items-center gap-2 text-green-400">
            <Check className="h-4 w-4" />
            <span>Attack mitigated successfully</span>
          </div>
        )}
      </div>
    </div>
  );
}

function StatsSection() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true });

  return (
    <section ref={ref} className="bg-white py-16 dark:bg-[#f8fafc]">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
          {stats.map((stat, index) => (
            <motion.div
              key={stat.label}
              initial={{ opacity: 0, y: 20 }}
              animate={isInView ? { opacity: 1, y: 0 } : {}}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="text-center"
            >
              <div className="text-4xl font-bold text-fortress-navy md:text-5xl">
                <AnimatedCounter value={stat.value} />
              </div>
              <div className="mt-2 text-sm text-fortress-gray md:text-base">
                {stat.label}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function FeaturesSection() {
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true });

  return (
    <section id="features" className="bg-fortress-navy py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-white md:text-4xl">
            Security That Actually Works
          </h2>
          <p className="mt-4 text-lg text-fortress-gray-light">
            Enterprise-grade protection without the enterprise complexity.
          </p>
        </div>

        <div
          ref={ref}
          className="mt-16 grid gap-8 md:grid-cols-2 lg:grid-cols-3"
        >
          {features.map((feature, index) => (
            <motion.div
              key={feature.title}
              initial={{ opacity: 0, y: 20 }}
              animate={isInView ? { opacity: 1, y: 0 } : {}}
              transition={{ duration: 0.5, delay: index * 0.1 }}
            >
              <Card className="h-full border-fortress-navy-light bg-fortress-navy-light/50">
                <CardContent className="p-6">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-fortress-green/20">
                    <feature.icon className="h-6 w-6 text-fortress-green" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold text-white">
                    {feature.title}
                  </h3>
                  <p className="mt-2 text-fortress-gray-light">
                    {feature.description}
                  </p>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function ComparisonSection() {
  return (
    <section className="bg-white py-24 dark:bg-[#f8fafc]">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-fortress-navy md:text-4xl">
            Why FortressWAF?
          </h2>
          <p className="mt-4 text-lg text-fortress-gray">
            The only WAF that puts security and developer experience first.
          </p>
        </div>

        <div className="mt-16 overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-fortress-navy-light">
                <th className="py-4 pr-4 text-left text-sm font-semibold text-fortress-navy">
                  Feature
                </th>
                <th className="px-4 text-center text-sm font-semibold text-fortress-green">
                  FortressWAF
                </th>
                <th className="px-4 text-center text-sm font-semibold text-fortress-navy">
                  Cloudflare WAF
                </th>
                <th className="px-4 text-center text-sm font-semibold text-fortress-navy">
                  Imperva
                </th>
                <th className="px-4 text-center text-sm font-semibold text-fortress-navy">
                  AppTrana
                </th>
              </tr>
            </thead>
            <tbody>
              {comparisonData.map((row, index) => (
                <tr
                  key={row.feature}
                  className={`border-b border-fortress-navy-light/20 ${
                    index % 2 === 0 ? "bg-fortress-navy/5" : ""
                  }`}
                >
                  <td className="py-4 pr-4 text-sm text-fortress-navy">
                    {row.feature}
                  </td>
                  <td className="px-4 text-center">
                    {typeof row.fortress === "boolean" ? (
                      row.fortress ? (
                        <Check className="mx-auto h-5 w-5 text-green-500" />
                      ) : (
                        <X className="mx-auto h-5 w-5 text-red-500" />
                      )
                    ) : (
                      <span className="text-sm font-medium text-fortress-green">
                        {row.fortress}
                      </span>
                    )}
                  </td>
                  <td className="px-4 text-center">
                    {typeof row.cloudflare === "boolean" ? (
                      row.cloudflare ? (
                        <Check className="mx-auto h-5 w-5 text-green-500" />
                      ) : (
                        <X className="mx-auto h-5 w-5 text-red-500" />
                      )
                    ) : (
                      <span className="text-sm text-fortress-gray">
                        {row.cloudflare}
                      </span>
                    )}
                  </td>
                  <td className="px-4 text-center">
                    {typeof row.imperva === "boolean" ? (
                      row.imperva ? (
                        <Check className="mx-auto h-5 w-5 text-green-500" />
                      ) : (
                        <X className="mx-auto h-5 w-5 text-red-500" />
                      )
                    ) : (
                      <span className="text-sm text-fortress-gray">
                        {row.imperva}
                      </span>
                    )}
                  </td>
                  <td className="px-4 text-center">
                    {typeof row.appTrana === "boolean" ? (
                      row.appTrana ? (
                        <Check className="mx-auto h-5 w-5 text-green-500" />
                      ) : (
                        <X className="mx-auto h-5 w-5 text-red-500" />
                      )
                    ) : (
                      <span className="text-sm text-fortress-gray">
                        {row.appTrana}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function PricingSection() {
  const [isAnnual, setIsAnnual] = useState(true);

  return (
    <section className="bg-fortress-navy py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-white md:text-4xl">
            Simple, Transparent Pricing
          </h2>
          <p className="mt-4 text-lg text-fortress-gray-light">
            Start free. Scale as you grow.
          </p>

          <div className="mt-8 flex items-center justify-center gap-4">
            <span
              className={`text-sm ${
                !isAnnual ? "text-white" : "text-fortress-gray"
              }`}
            >
              Monthly
            </span>
            <button
              onClick={() => setIsAnnual(!isAnnual)}
              className={`relative h-6 w-11 rounded-full transition-colors ${
                isAnnual ? "bg-fortress-green" : "bg-fortress-navy-light"
              }`}
            >
              <span
                className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${
                  isAnnual ? "translate-x-5" : "translate-x-0"
                }`}
              />
            </button>
            <span
              className={`text-sm ${
                isAnnual ? "text-white" : "text-fortress-gray"
              }`}
            >
              Annual{" "}
              <Badge variant="success" className="ml-1">
                Save 20%
              </Badge>
            </span>
          </div>
        </div>

        <div className="mt-16 grid gap-8 md:grid-cols-2 lg:grid-cols-4">
          {pricingTiers.map((tier, index) => (
            <motion.div
              key={tier.name}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
            >
              <Card
                className={`relative h-full ${
                  tier.popular
                    ? "border-fortress-green bg-fortress-navy-light"
                    : ""
                }`}
              >
                {tier.popular && (
                  <div className="absolute -top-4 left-1/2 -translate-x-1/2">
                    <Badge className="bg-fortress-green">Most Popular</Badge>
                  </div>
                )}
                <CardContent className="p-6">
                  <h3 className="text-lg font-semibold text-white">
                    {tier.name}
                  </h3>
                  <p className="mt-2 text-sm text-fortress-gray-light">
                    {tier.description}
                  </p>
                  <div className="mt-6">
                    {tier.price !== null ? (
                      <div className="flex items-baseline">
                        <span className="text-4xl font-bold text-white">
                          ${isAnnual ? Math.round(tier.price * 0.8) : tier.price}
                        </span>
                        <span className="ml-2 text-fortress-gray">/mo</span>
                      </div>
                    ) : (
                      <div className="text-3xl font-bold text-white">
                        Custom
                      </div>
                    )}
                    {isAnnual && tier.price !== null && (
                      <p className="mt-1 text-xs text-fortress-gray">
                        Billed annually
                      </p>
                    )}
                  </div>
                  <ul className="mt-6 space-y-3">
                    {tier.features.map((feature) => (
                      <li
                        key={feature}
                        className="flex items-start gap-2 text-sm text-fortress-gray-light"
                      >
                        <Check className="mt-0.5 h-4 w-4 flex-shrink-0 text-fortress-green" />
                        {feature}
                      </li>
                    ))}
                  </ul>
                  <Button
                    className={`mt-6 w-full ${
                      tier.popular ? "" : "variant=outline"
                    }`}
                    variant={tier.popular ? "default" : "outline"}
                    asChild
                  >
                    <Link href={tier.href}>{tier.cta}</Link>
                  </Button>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function TestimonialsSection() {
  return (
    <section className="bg-white py-24 dark:bg-[#f8fafc]">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-fortress-navy md:text-4xl">
            Trusted by Security Teams
          </h2>
          <p className="mt-4 text-lg text-fortress-gray">
            See what our customers have to say.
          </p>
        </div>

        <div className="mt-16 grid gap-8 md:grid-cols-3">
          {testimonials.map((testimonial, index) => (
            <motion.div
              key={testimonial.author}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
            >
              <Card className="h-full">
                <CardContent className="p-6">
                  <div className="flex gap-1">
                    {[...Array(5)].map((_, i) => (
                      <Star
                        key={i}
                        className="h-4 w-4 fill-amber-400 text-amber-400"
                      />
                    ))}
                  </div>
                  <p className="mt-4 text-fortress-navy">
                    &ldquo;{testimonial.quote}&rdquo;
                  </p>
                  <div className="mt-6">
                    <p className="font-semibold text-fortress-navy">
                      {testimonial.author}
                    </p>
                    <p className="text-sm text-fortress-gray">
                      {testimonial.title}, {testimonial.company}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

function CTASection() {
  return (
    <section className="bg-fortress-green py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          <h2 className="text-3xl font-bold text-white md:text-4xl">
            Ready to secure your applications?
          </h2>
          <p className="mt-4 text-lg text-white/80">
            Join thousands of teams protecting their infrastructure with
            FortressWAF.
          </p>
          <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Button
              size="lg"
              className="bg-white text-fortress-green hover:bg-white/90"
              asChild
            >
              <Link href="/contact">
                Start Free <ArrowRight className="ml-2 h-5 w-5" />
              </Link>
            </Button>
            <Button
              size="lg"
              variant="outline"
              className="border-white text-white hover:bg-white/10"
              asChild
            >
              <a
                href="https://github.com"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Github className="mr-2 h-5 w-5" />
                View on GitHub
              </a>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function HomePage() {
  return (
    <div className="bg-fortress-navy">
      {/* Hero Section */}
      <section className="relative min-h-screen flex items-center pt-16 overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-b from-fortress-navy via-fortress-navy to-fortress-navy-light" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-fortress-green/20 via-transparent to-transparent" />

        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24">
          <div className="grid gap-16 lg:grid-cols-2 lg:gap-16 items-center">
            <div className="flex flex-col">
              <div className="inline-flex items-center gap-2 rounded-full border border-fortress-green/30 bg-fortress-green/10 px-3 py-1 text-sm text-fortress-green-light w-fit">
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-fortress-green opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-fortress-green"></span>
                </span>
                Now in General Availability
              </div>

              <h1 className="mt-6 text-4xl font-bold text-white sm:text-5xl md:text-6xl leading-tight">
                Enterprise WAF Security.{" "}
                <span className="text-fortress-green">Developer Simplicity.</span>
              </h1>

              <p className="mt-6 text-lg text-fortress-gray-light max-w-xl">
                Stop web attacks in real time. Deploy in under 30 minutes.
                Open-core, no vendor lock-in.
              </p>

              <div className="mt-8 flex flex-col sm:flex-row gap-4">
                <Button
                  size="lg"
                  className="bg-fortress-green text-white hover:bg-fortress-green-light"
                  asChild
                >
                  <Link href="/contact">
                    Start Free <ArrowRight className="ml-2 h-5 w-5" />
                  </Link>
                </Button>
                <Button
                  size="lg"
                  variant="outline"
                  className="border-fortress-gray-light text-fortress-gray-light hover:bg-fortress-navy-light hover:text-white"
                  asChild
                >
                  <a
                    href="https://github.com"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Github className="mr-2 h-5 w-5" />
                    View on GitHub
                  </a>
                </Button>
              </div>

              <div className="mt-8 flex flex-wrap items-center gap-6 text-sm text-fortress-gray">
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-fortress-green" />
                  <span>OWASP CRS 4.0</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-fortress-green" />
                  <span>SOC2 Ready</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-fortress-green" />
                  <span>PCI DSS</span>
                </div>
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-fortress-green" />
                  <span>Open Source Core</span>
                </div>
              </div>
            </div>

            <div className="lg:pl-8">
              <TerminalAnimation />
            </div>
          </div>
        </div>
      </section>

      {/* Trust badges */}
      <div className="border-y border-fortress-navy-light bg-fortress-navy/50 py-8">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p className="text-center text-sm text-fortress-gray">
            Trusted by security teams at
          </p>
          <div className="mt-4 flex flex-wrap items-center justify-center gap-8 opacity-60">
            {["TechCorp", "SecureBank", "CloudFirst", "DataFlow", "FinServ"].map(
              (company) => (
                <span
                  key={company}
                  className="text-lg font-semibold text-fortress-gray-light"
                >
                  {company}
                </span>
              )
            )}
          </div>
        </div>
      </div>

      <StatsSection />
      <FeaturesSection />
      <ComparisonSection />
      <PricingSection />
      <TestimonialsSection />
      <CTASection />
    </div>
  );
}
