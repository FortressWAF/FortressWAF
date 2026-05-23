import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Calendar, Clock, Share2 } from "lucide-react";

const blogPosts: Record<
  string,
  {
    title: string;
    date: string;
    readTime: string;
    category: string;
    content: React.ReactNode;
  }
> = {
  "how-fortresswaf-stopped-50gbps-ddos-attack": {
    title: "How FortressWAF Stopped a 50Gbps DDoS Attack",
    date: "March 15, 2025",
    readTime: "8 min read",
    category: "Case Study",
    content: (
      <>
        <p className="lead text-xl text-fortress-gray-light">
          On February 3rd, 2025, one of our enterprise customers—a major
          e-commerce platform in Europe—faced the largest DDoS attack in their
          history. Here's how FortressWAF handled it.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          The Attack
        </h2>
        <p className="text-fortress-gray-light">
          At 14:23 UTC, our anomaly detection engine flagged unusual traffic
          patterns targeting the customer's public API endpoints. Initial volume
          was around 2Gbps, but within 15 minutes, the attack scaled to 50Gbps
          using a UDP reflection attack vector exploiting open DNS resolvers.
        </p>

        <div className="bg-fortress-navy-light rounded-lg p-6 my-8 border border-fortress-navy-light">
          <p className="text-sm text-fortress-gray font-mono">
            Attack Vector: UDP Flood (DNS Amplification)
            <br />
            Peak Volume: 50.2 Gbps
            <br />
            Peak PPS: 42.1 Million
            <br />
            Duration: 47 minutes
            <br />
            Attackers: 12,847 unique IPs
          </p>
        </div>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          Our Response
        </h2>
        <p className="text-fortress-gray-light">
          Within 30 seconds of detection, our ML models classified the traffic
          as malicious and automatically deployed mitigation rules. The anycast
          network distributed traffic across 23 PoPs, absorbing the volumetric
          attack at the edge.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          The Outcome
        </h2>
        <p className="text-fortress-gray-light">
          Zero downtime. Zero dropped legitimate requests. The customer's
          platform remained fully operational throughout the attack, while our
          systems processed and blocked 2.3 billion malicious packets.
        </p>

        <div className="bg-fortress-green/20 border border-fortress-green rounded-lg p-6 my-8">
          <p className="text-fortress-green font-medium">
            "We had zero indication anything was happening. Our monitoring
            dashboard showed normal traffic. FortressWAF handled everything
            silently in the background."
          </p>
          <p className="text-fortress-gray-light text-sm mt-2">
            — CTO, European E-commerce Platform
          </p>
        </div>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          Technical Deep Dive
        </h2>
        <p className="text-fortress-gray-light">
          The attack was a multi-vector UDP reflection using DNS, NTP, and
          Memcached protocols. Our ML engine identified the attack signature
          within the first 30 seconds by analyzing packet characteristics that
          differed from legitimate traffic patterns:
        </p>

        <ul className="list-disc list-inside text-fortress-gray-light mt-4 space-y-2">
          <li>Unusually high packet rate from single source ranges</li>
          <li>DNS responses exceeding normal query ratios by 40x</li>
          <li>Geographic distribution not matching normal traffic patterns</li>
          <li>TTL values inconsistent with legitimate DNS resolvers</li>
        </ul>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          Lessons Learned
        </h2>
        <p className="text-fortress-gray-light">
          This attack demonstrated the importance of having automated,
          ML-driven mitigation. Traditional rule-based WAFs would have
          required manual rule updates to handle novel attack vectors. Our
          system adapted in real-time without human intervention.
        </p>
      </>
    ),
  },
  "owasp-top-10-2024-update": {
    title: "OWASP Top 10: 2024 Update and How to Protect Against Each",
    date: "February 28, 2025",
    readTime: "12 min read",
    category: "Security",
    content: (
      <>
        <p className="lead text-xl text-fortress-gray-light">
          The OWASP Foundation has released the 2024 update to the OWASP Top 10.
          This guide covers each vulnerability class and how FortressWAF protects
          against them.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A01:2021 - Broken Access Control
        </h2>
        <p className="text-fortress-gray-light">
          Access control vulnerabilities remain the most critical web security
          risk. Broken access control occurs when users can act outside their
          intended permissions.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          API Security module validates authorization tokens, enforces
          role-based access control, and detects horizontal privilege escalation
          attempts.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A02:2021 - Cryptographic Failures
        </h2>
        <p className="text-fortress-gray-light">
          Formerly known as Sensitive Data Exposure, this category focuses on
          failures related to cryptography which often lead to exposure of
          sensitive data.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> We
          detect and block endpoints that expose sensitive data in responses,
          and provide headers for HSTS, CSP, and other security headers.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A03:2021 - Injection
        </h2>
        <p className="text-fortress-gray-light">
          SQL, NoSQL, OS, and LDAP injection remain prevalent. User-supplied
          data is not properly validated, filtered, or sanitized.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          core detection engine includes rules for all injection types,
          validated against the OWASP CRS. ML models provide additional
          zero-day protection.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A04:2021 - Insecure Design
        </h2>
        <p className="text-fortress-gray-light">
          A new category for 2021, focusing on risks related to design and
          architectural weaknesses.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          threat intelligence feeds and virtual patching capabilities help
          mitigate insecure design patterns until they can be properly remediated.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A05:2021 - Security Misconfiguration
        </h2>
        <p className="text-fortress-gray-light">
          Missing hardening, overly permissive configurations, and verbose
          error messages are common security misconfigurations.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> We
          provide automated security header configuration, detect information
          leakage in error responses, and scan for common misconfigurations.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A06:2021 - Vulnerable and Outdated Components
        </h2>
        <p className="text-fortress-gray-light">
          You are likely vulnerable if you don't know the versions of all
          components you use, or if software is outdated or unsupported.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          virtual patching feature provides immediate protection for known
          vulnerabilities while you work on permanent remediation.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A07:2021 - Identification and Authentication Failures
        </h2>
        <p className="text-fortress-gray-light">
          Confirmation of the user's identity, authentication, and session
          management is critical to protect against authentication-related attacks.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          rate limiting and bot management detect credential stuffing attacks,
          while our session management features help enforce secure session handling.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A08:2021 - Software and Data Integrity Failures
        </h2>
        <p className="text-fortress-gray-light">
          Software and data integrity failures relate to code and infrastructure
          that does not protect against integrity violations.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> We
          detect supply chain attacks and untrusted content inclusion, while
          providing CI/CD security integrations.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A09:2021 - Security Logging and Monitoring Failures
        </h2>
        <p className="text-fortress-gray-light">
          Without logging and monitoring, breaches cannot be detected.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          real-time dashboard provides comprehensive logging and alerting,
          with integrations to SIEM platforms and custom webhooks.
        </p>

        <h2 className="text-2xl font-bold text-white mt-8 mb-4">
          A10:2021 - Server-Side Request Forgery (SSRF)
        </h2>
        <p className="text-fortress-gray-light">
          SSRF flaws occur when a web application fetches a remote resource
          without validating the user-supplied URL.
        </p>
        <p className="text-fortress-gray-light mt-4">
          <strong className="text-white">FortressWAF Protection:</strong> Our
          outbound request validation blocks SSRF attempts by validating URLs
          against allowlists and blocking known malicious patterns.
        </p>

        <div className="bg-fortress-navy-light rounded-lg p-6 my-8 border border-fortress-navy-light">
          <p className="text-fortress-gray-light">
            <strong className="text-white">Protection Summary:</strong> FortressWAF
            provides comprehensive protection against all OWASP Top 10
            vulnerability classes through a combination of rule-based detection,
            ML-powered analysis, and real-time threat intelligence.
          </p>
        </div>
      </>
    ),
  },
};

interface PageProps {
  params: { slug: string };
}

export default function BlogPostPage({ params }: PageProps) {
  const post = blogPosts[params.slug];

  if (!post) {
    return (
      <div className="min-h-screen bg-fortress-navy pt-16 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white">Post not found</h1>
          <Link href="/blog">
            <Button className="mt-4" variant="outline">
              Back to Blog
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-fortress-navy pt-16">
      <article className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8 py-24">
        <Link
          href="/blog"
          className="inline-flex items-center text-sm text-fortress-gray hover:text-white mb-8"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Blog
        </Link>

        <Badge variant="secondary" className="mb-4">
          {post.category}
        </Badge>

        <h1 className="text-4xl font-bold text-white md:text-5xl leading-tight">
          {post.title}
        </h1>

        <div className="flex items-center gap-6 mt-6 text-sm text-fortress-gray">
          <div className="flex items-center gap-1">
            <Calendar className="h-4 w-4" />
            {post.date}
          </div>
          <div className="flex items-center gap-1">
            <Clock className="h-4 w-4" />
            {post.readTime}
          </div>
        </div>

        <div className="mt-8 flex items-center gap-2">
          <Button variant="outline" size="sm" className="text-fortress-gray-light">
            <Share2 className="h-4 w-4 mr-2" />
            Share
          </Button>
        </div>

        <div className="mt-12 prose prose-invert prose-lg max-w-none">
          {post.content}
        </div>

        <div className="mt-16 border-t border-fortress-navy-light pt-8">
          <h3 className="text-lg font-semibold text-white mb-4">
            Related Articles
          </h3>
          <div className="grid gap-4 md:grid-cols-2">
            {Object.entries(blogPosts)
              .filter(([slug]) => slug !== params.slug)
              .slice(0, 2)
              .map(([slug, relatedPost]) => (
                <Link
                  key={slug}
                  href={`/blog/${slug}`}
                  className="text-fortress-green hover:underline"
                >
                  {relatedPost.title}
                </Link>
              ))}
          </div>
        </div>
      </article>
    </div>
  );
}
