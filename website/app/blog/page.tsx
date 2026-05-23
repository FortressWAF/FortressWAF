import Link from "next/link";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Calendar, Clock, ArrowRight } from "lucide-react";

const blogPosts = [
  {
    slug: "how-fortresswaf-stopped-50gbps-ddos-attack",
    title: "How FortressWAF Stopped a 50Gbps DDoS Attack",
    excerpt:
      "A detailed breakdown of how our anycast network and ML-based rate limiting neutralized the largest DDoS attack in our customer's history.",
    date: "March 15, 2025",
    readTime: "8 min read",
    category: "Case Study",
  },
  {
    slug: "owasp-top-10-2024-update",
    title: "OWASP Top 10: 2024 Update and How to Protect Against Each",
    excerpt:
      "The 2024 OWASP Top 10 list is here. Learn what's new and how FortressWAF protects against each vulnerability class.",
    date: "February 28, 2025",
    readTime: "12 min read",
    category: "Security",
  },
  {
    slug: "kubernetes-waf-deployment-guide",
    title: "Deploying FortressWAF on Kubernetes: A Complete Guide",
    excerpt:
      "Step-by-step tutorial for deploying FortressWAF as a Kubernetes sidecar or ingress controller with Helm charts.",
    date: "February 10, 2025",
    readTime: "15 min read",
    category: "Tutorial",
  },
];

export default function BlogPage() {
  return (
    <div className="min-h-screen bg-fortress-navy pt-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24">
        <div className="text-center mb-16">
          <Badge className="mb-4">Blog</Badge>
          <h1 className="text-4xl font-bold text-white md:text-5xl">
            Security Insights
          </h1>
          <p className="mt-4 text-lg text-fortress-gray-light max-w-2xl mx-auto">
            Expert analysis, tutorials, and best practices for web application
            security.
          </p>
        </div>

        <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {blogPosts.map((post) => (
            <Link key={post.slug} href={`/blog/${post.slug}`}>
              <Card className="h-full hover:border-fortress-green transition-colors cursor-pointer">
                <CardHeader>
                  <Badge variant="secondary" className="w-fit mb-2">
                    {post.category}
                  </Badge>
                  <CardTitle className="text-white leading-tight">
                    {post.title}
                  </CardTitle>
                  <CardDescription className="text-fortress-gray-light line-clamp-2">
                    {post.excerpt}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-4 text-sm text-fortress-gray">
                    <div className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      {post.date}
                    </div>
                    <div className="flex items-center gap-1">
                      <Clock className="h-4 w-4" />
                      {post.readTime}
                    </div>
                  </div>
                  <div className="mt-4 flex items-center text-fortress-green text-sm font-medium">
                    Read more <ArrowRight className="ml-1 h-4 w-4" />
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        <div className="mt-16 text-center">
          <p className="text-fortress-gray">
            Want to stay updated?{" "}
            <Link href="#" className="text-fortress-green hover:underline">
              Subscribe to our newsletter
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
