"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Check, X, ArrowRight, HelpCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Separator } from "@/components/ui/separator";

const pricingTiers = [
  {
    name: "Community",
    price: 0,
    description: "For individual developers and small projects.",
    features: [
      { name: "3 rulesets", included: true },
      { name: "10K requests/month", included: true },
      { name: "Community support", included: true },
      { name: "Basic analytics", included: true },
      { name: "Self-hosted", included: true },
      { name: "API access", included: false },
      { name: "ML detection", included: false },
      { name: "Priority support", included: false },
      { name: "SOC 2 reports", included: false },
    ],
    cta: "Get Started",
    href: "/contact",
  },
  {
    name: "Starter",
    price: 149,
    description: "For growing applications and startups.",
    features: [
      { name: "Unlimited rulesets", included: true },
      { name: "1M requests/month", included: true },
      { name: "Email support", included: true },
      { name: "Advanced analytics", included: true },
      { name: "API access", included: true },
      { name: "Basic ML detection", included: true },
      { name: "Priority support", included: false },
      { name: "SOC 2 reports", included: false },
      { name: "Custom SLA", included: false },
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
      { name: "Unlimited requests", included: true },
      { name: "Priority support", included: true },
      { name: "Advanced ML detection", included: true },
      { name: "Multi-tenant management", included: true },
      { name: "Custom rules", included: true },
      { name: "SOC 2 reports", included: true },
      { name: "Dedicated support", included: false },
      { name: "On-premise deployment", included: false },
      { name: "White-label options", included: false },
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
      { name: "Everything in Professional", included: true },
      { name: "Dedicated support engineer", included: true },
      { name: "Custom SLA", included: true },
      { name: "On-premise deployment", included: true },
      { name: "White-label options", included: true },
      { name: "Advanced integrations", included: true },
      { name: "Compliance reports", included: true },
      { name: "Unlimited rulesets", included: true },
      { name: "Unlimited requests", included: true },
    ],
    cta: "Contact Sales",
    href: "/contact",
  },
];

const faqs = [
  {
    question: "What happens if I exceed my request limit?",
    answer:
      "We'll notify you when you reach 80% of your limit. You can upgrade at any time, or requests will be temporarily queued during overages. We never silently drop traffic.",
  },
  {
    question: "Can I switch plans at any time?",
    answer:
      "Yes, you can upgrade or downgrade your plan at any time. Upgrades take effect immediately, and we'll prorate the difference. Downgrades take effect at the start of your next billing cycle.",
  },
  {
    question: "Do you offer discounts for startups or non-profits?",
    answer:
      "Absolutely. We offer 50% off for startups under 2 years old with less than $1M in funding, and free access for non-profit organizations. Contact our sales team for details.",
  },
  {
    question: "What's included in the self-hosted option?",
    answer:
      "All plans include the option to self-host using Docker, Kubernetes, or bare metal. You'll get access to our container images and Helm charts. Enterprise customers get dedicated support for on-premise deployments.",
  },
  {
    question: "How does the open-source core work?",
    answer:
      "Our core detection engine is open-source under the Apache 2.0 license. You can inspect, audit, and run it independently. The cloud management plane and advanced ML features are proprietary add-ons.",
  },
];

export default function PricingPage() {
  const [isAnnual, setIsAnnual] = useState(true);

  return (
    <div className="min-h-screen bg-fortress-navy pt-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center">
          <Badge className="mb-4">Pricing</Badge>
          <h1 className="text-4xl font-bold text-white md:text-5xl">
            Simple, Transparent Pricing
          </h1>
          <p className="mt-4 text-lg text-fortress-gray-light max-w-2xl mx-auto">
            Start free with our community edition. Scale as you grow with
            plans that won't surprise you with hidden fees.
          </p>

          <div className="mt-8 flex items-center justify-center gap-4">
            <span
              className={`text-sm ${!isAnnual ? "text-white" : "text-fortress-gray"}`}
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
              className={`text-sm ${isAnnual ? "text-white" : "text-fortress-gray"}`}
            >
              Annual{" "}
              <Badge variant="success" className="ml-1">
                Save 20%
              </Badge>
            </span>
          </div>
        </div>

        <div className="mt-16 grid gap-8 md:grid-cols-2 lg:grid-cols-4">
          {pricingTiers.map((tier) => (
            <Card
              key={tier.name}
              className={`relative ${
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
              <CardHeader>
                <CardTitle className="text-white">{tier.name}</CardTitle>
                <p className="text-sm text-fortress-gray-light">
                  {tier.description}
                </p>
              </CardHeader>
              <CardContent>
                <div className="mb-6">
                  {tier.price !== null ? (
                    <div className="flex items-baseline">
                      <span className="text-4xl font-bold text-white">
                        ${isAnnual ? Math.round(tier.price * 0.8) : tier.price}
                      </span>
                      <span className="ml-2 text-fortress-gray">/mo</span>
                    </div>
                  ) : (
                    <div className="text-3xl font-bold text-white">Custom</div>
                  )}
                  {isAnnual && tier.price !== null && tier.price > 0 && (
                    <p className="mt-1 text-xs text-fortress-gray">
                      Billed annually (${Math.round(tier.price * 0.8 * 12)}/year)
                    </p>
                  )}
                </div>
                <Button
                  className={`w-full ${
                    tier.popular ? "" : "variant=outline"
                  }`}
                  variant={tier.popular ? "default" : "outline"}
                  asChild
                >
                  <Link href={tier.href}>{tier.cta}</Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>

        <Separator className="my-24 bg-fortress-navy-light" />

        <div className="mb-16">
          <h2 className="text-2xl font-bold text-white text-center mb-8">
            Compare Plans
          </h2>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-fortress-navy-light">
                  <th className="py-4 pr-4 text-left text-sm font-semibold text-fortress-gray-light">
                    Feature
                  </th>
                  <th className="px-4 text-center text-sm font-semibold text-white">
                    Community
                  </th>
                  <th className="px-4 text-center text-sm font-semibold text-white">
                    Starter
                  </th>
                  <th className="px-4 text-center text-sm font-semibold text-fortress-green">
                    Professional
                  </th>
                  <th className="px-4 text-center text-sm font-semibold text-white">
                    Enterprise
                  </th>
                </tr>
              </thead>
              <tbody>
                {[
                  ["OWASP Top 10 Protection", true, true, true, true],
                  ["SQL Injection Prevention", true, true, true, true],
                  ["XSS Protection", true, true, true, true],
                  ["Rate Limiting", true, true, true, true],
                  ["API Security", false, true, true, true],
                  ["Bot Management", false, true, true, true],
                  ["ML Zero-Day Detection", false, "Basic", true, true],
                  ["Self-Hosted Option", true, true, true, true],
                  ["Multi-Tenant Management", false, false, true, true],
                  ["Custom Rules", false, false, true, true],
                  ["SOC 2 Reports", false, false, true, true],
                  ["Dedicated Support", false, false, false, true],
                  ["White-Label", false, false, false, true],
                ].map(([feature, community, starter, professional, enterprise]) => (
                  <tr
                    key={feature as string}
                    className="border-b border-fortress-navy-light/20"
                  >
                    <td className="py-4 pr-4 text-sm text-fortress-gray-light">
                      {feature as string}
                    </td>
                    <td className="px-4 text-center">
                      {typeof community === "boolean" ? (
                        community ? (
                          <Check className="mx-auto h-5 w-5 text-green-500" />
                        ) : (
                          <X className="mx-auto h-5 w-5 text-red-500" />
                        )
                      ) : (
                        <span className="text-xs text-fortress-gray-light">
                          {community as string}
                        </span>
                      )}
                    </td>
                    <td className="px-4 text-center">
                      {typeof starter === "boolean" ? (
                        starter ? (
                          <Check className="mx-auto h-5 w-5 text-green-500" />
                        ) : (
                          <X className="mx-auto h-5 w-5 text-red-500" />
                        )
                      ) : (
                        <span className="text-xs text-fortress-gray-light">
                          {starter as string}
                        </span>
                      )}
                    </td>
                    <td className="px-4 text-center">
                      {typeof professional === "boolean" ? (
                        professional ? (
                          <Check className="mx-auto h-5 w-5 text-green-500" />
                        ) : (
                          <X className="mx-auto h-5 w-5 text-red-500" />
                        )
                      ) : (
                        <span className="text-xs text-fortress-green">
                          {professional as string}
                        </span>
                      )}
                    </td>
                    <td className="px-4 text-center">
                      {typeof enterprise === "boolean" ? (
                        enterprise ? (
                          <Check className="mx-auto h-5 w-5 text-green-500" />
                        ) : (
                          <X className="mx-auto h-5 w-5 text-red-500" />
                        )
                      ) : (
                        <span className="text-xs text-fortress-gray-light">
                          {enterprise as string}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <Separator className="my-24 bg-fortress-navy-light" />

        <div className="mb-16">
          <h2 className="text-2xl font-bold text-white text-center mb-8">
            Frequently Asked Questions
          </h2>
          <div className="mx-auto max-w-3xl">
            <Accordion type="single" collapsible className="w-full">
              {faqs.map((faq, index) => (
                <AccordionItem key={index} value={`item-${index}`}>
                  <AccordionTrigger className="text-left text-white">
                    {faq.question}
                  </AccordionTrigger>
                  <AccordionContent className="text-fortress-gray-light">
                    {faq.answer}
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </div>
        </div>

        <div className="text-center bg-fortress-navy-light rounded-2xl p-12">
          <h3 className="text-2xl font-bold text-white">
            Still have questions?
          </h3>
          <p className="mt-2 text-fortress-gray-light">
            Our team is here to help you find the perfect plan for your needs.
          </p>
          <Button className="mt-6" asChild>
            <Link href="/contact">
              Contact Sales <ArrowRight className="ml-2 h-5 w-5" />
            </Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
