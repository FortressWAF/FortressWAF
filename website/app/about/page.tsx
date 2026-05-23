import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Github, Twitter, Linkedin, Shield, Users, Globe, Heart } from "lucide-react";

const teamMembers = [
  {
    name: "Alexandra Chen",
    role: "CEO & Co-founder",
    bio: "Former security lead at Stripe. 15 years in enterprise security.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
  {
    name: "Marcus Williams",
    role: "CTO & Co-founder",
    bio: "Core contributor to ModSecurity. Built WAF infrastructure at Cloudflare.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
  {
    name: "Priya Sharma",
    role: "VP of Engineering",
    bio: "Led security product at GitHub. Expert in threat detection ML.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
  {
    name: "David Kim",
    role: "Head of Security Research",
    bio: "Discovered 50+ CVEs. Former NSA red team. OWASP contributor.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
  {
    name: "Sarah Okonkwo",
    role: "VP of Sales",
    bio: "10 years enterprise security sales. Built MSSP channel at Imperva.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
  {
    name: "James Rodriguez",
    role: "Head of Customer Success",
    bio: "Former CISO at Fortune 500. Expert in compliance and security architecture.",
    social: {
      twitter: "#",
      linkedin: "#",
      github: "#",
    },
  },
];

const values = [
  {
    icon: Shield,
    title: "Security First",
    description:
      "We believe security should never be an afterthought. Every feature we build starts with security considerations.",
  },
  {
    icon: Globe,
    title: "Open Source",
    description:
      "Our core engine is open source because transparency builds trust. Audit our code, contribute, and help us improve.",
  },
  {
    icon: Users,
    title: "Customer Success",
    description:
      "Your security is our success. We provide 24/7 support and work alongside you during critical incidents.",
  },
  {
    icon: Heart,
    title: "Community",
    description:
      "We're building a community of security professionals who share knowledge and best practices.",
  },
];

export default function AboutPage() {
  return (
    <div className="min-h-screen bg-fortress-navy pt-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center mb-20">
          <Badge className="mb-4">About Us</Badge>
          <h1 className="text-4xl font-bold text-white md:text-5xl">
            Built by Security Veterans
          </h1>
          <p className="mt-6 text-lg text-fortress-gray-light max-w-3xl mx-auto">
            We created FortressWAF because we were tired of WAFs that were either
            too expensive, too complicated, or too closed. We believe enterprise
            security should be accessible, transparent, and developer-friendly.
          </p>
        </div>

        <div className="grid gap-16 mb-20">
          <div className="grid gap-12 md:grid-cols-2 items-center">
            <div>
              <h2 className="text-2xl font-bold text-white mb-4">
                Our Mission
              </h2>
              <p className="text-fortress-gray-light leading-relaxed">
                Web applications are the backbone of modern business, yet they
                remain vulnerable to attacks that cause data breaches, downtime,
                and reputational damage. We founded FortressWAF to democratize
                enterprise-grade web security.
              </p>
              <p className="mt-4 text-fortress-gray-light leading-relaxed">
                Our mission is to make it easy for any organization—from startups
                to enterprises—to protect their web applications without vendor
                lock-in, excessive complexity, or prohibitive costs.
              </p>
            </div>
            <div className="bg-fortress-navy-light rounded-2xl p-8 border border-fortress-navy-light">
              <div className="grid grid-cols-2 gap-8">
                <div className="text-center">
                  <div className="text-4xl font-bold text-fortress-green">2M+</div>
                  <div className="mt-2 text-sm text-fortress-gray-light">
                    Requests protected daily
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-4xl font-bold text-fortress-green">500+</div>
                  <div className="mt-2 text-sm text-fortress-gray-light">
                    Enterprise customers
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-4xl font-bold text-fortress-green">50+</div>
                  <div className="mt-2 text-sm text-fortress-gray-light">
                    Team members globally
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-4xl font-bold text-fortress-green">99.9%</div>
                  <div className="mt-2 text-sm text-fortress-gray-light">
                    Customer satisfaction
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="mb-20">
          <h2 className="text-2xl font-bold text-white text-center mb-12">
            Our Values
          </h2>
          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-4">
            {values.map((value) => (
              <Card key={value.title} className="bg-fortress-navy-light border-fortress-navy-light">
                <CardContent className="p-6 text-center">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-fortress-green/20 mx-auto">
                    <value.icon className="h-6 w-6 text-fortress-green" />
                  </div>
                  <h3 className="mt-4 text-lg font-semibold text-white">
                    {value.title}
                  </h3>
                  <p className="mt-2 text-sm text-fortress-gray-light">
                    {value.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>

        <div className="mb-20">
          <h2 className="text-2xl font-bold text-white text-center mb-12">
            Meet the Team
          </h2>
          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
            {teamMembers.map((member) => (
              <Card
                key={member.name}
                className="bg-fortress-navy-light border-fortress-navy-light"
              >
                <CardContent className="p-6">
                  <div className="flex items-center gap-4">
                    <div className="h-16 w-16 rounded-full bg-fortress-navy flex items-center justify-center text-2xl font-bold text-fortress-green">
                      {member.name
                        .split(" ")
                        .map((n) => n[0])
                        .join("")}
                    </div>
                    <div>
                      <h3 className="font-semibold text-white">{member.name}</h3>
                      <p className="text-sm text-fortress-gray-light">
                        {member.role}
                      </p>
                    </div>
                  </div>
                  <p className="mt-4 text-sm text-fortress-gray-light">
                    {member.bio}
                  </p>
                  <div className="mt-4 flex gap-3">
                    <a
                      href={member.social.twitter}
                      className="text-fortress-gray hover:text-white transition-colors"
                    >
                      <Twitter className="h-4 w-4" />
                    </a>
                    <a
                      href={member.social.linkedin}
                      className="text-fortress-gray hover:text-white transition-colors"
                    >
                      <Linkedin className="h-4 w-4" />
                    </a>
                    <a
                      href={member.social.github}
                      className="text-fortress-gray hover:text-white transition-colors"
                    >
                      <Github className="h-4 w-4" />
                    </a>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>

        <div className="bg-fortress-navy-light rounded-2xl p-12 border border-fortress-navy-light">
          <div className="text-center max-w-3xl mx-auto">
            <h2 className="text-2xl font-bold text-white">
              Open Source Commitment
            </h2>
            <p className="mt-4 text-fortress-gray-light">
              Our core detection engine is fully open source under the Apache 2.0
              license. We believe in transparency and giving back to the security
              community. Our GitHub repository has over 15,000 stars and 200+
              contributors.
            </p>
            <p className="mt-4 text-fortress-gray-light">
              We actively contribute to OWASP projects, publish security research,
              and sponsor security conferences worldwide. Security is a shared
              responsibility, and we're committed to making the internet safer
              for everyone.
            </p>
            <div className="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
              <a
                href="https://github.com"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-4 py-2 text-sm font-medium text-fortress-navy hover:bg-fortress-gray-light"
              >
                <Github className="h-5 w-5" />
                View on GitHub
              </a>
              <Link
                href="/contact"
                className="inline-flex items-center justify-center gap-2 rounded-md border border-fortress-gray-light px-4 py-2 text-sm font-medium text-fortress-gray-light hover:bg-fortress-navy-light hover:text-white"
              >
                Join our Community
              </Link>
            </div>
          </div>
        </div>

        <div className="mt-16 text-center">
          <h2 className="text-2xl font-bold text-white mb-4">
            Join Our Mission
          </h2>
          <p className="text-fortress-gray-light max-w-2xl mx-auto">
            We're always looking for talented people who are passionate about
            security and want to make a difference. Check out our open positions.
          </p>
          <div className="mt-6">
            <Link
              href="/contact"
              className="inline-flex items-center justify-center rounded-md bg-fortress-green px-6 py-2 text-sm font-medium text-white hover:bg-fortress-green-light"
            >
              View Open Positions
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
