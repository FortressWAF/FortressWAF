"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/use-toast";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Mail, MessageSquare, Send } from "lucide-react";

const formSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  email: z.string().email("Please enter a valid email address"),
  company: z.string().optional(),
  message: z.string().min(10, "Message must be at least 10 characters"),
});

type FormData = z.infer<typeof formSchema>;

export default function ContactPage() {
  const { toast } = useToast();
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(formSchema),
  });

  const onSubmit = async (data: FormData) => {
    setIsSubmitting(true);
    await new Promise((resolve) => setTimeout(resolve, 1500));
    setIsSubmitting(false);
    toast({
      title: "Message sent!",
      description: "We'll get back to you within 24 hours.",
      variant: "default",
    });
    reset();
  };

  return (
    <div className="min-h-screen bg-fortress-navy pt-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-24">
        <div className="grid gap-16 lg:grid-cols-2">
          <div>
            <div className="inline-flex items-center gap-2 rounded-full border border-fortress-green/30 bg-fortress-green/10 px-3 py-1 text-sm text-fortress-green-light w-fit mb-4">
              <MessageSquare className="h-4 w-4" />
              Get in Touch
            </div>
            <h1 className="text-4xl font-bold text-white md:text-5xl">
              Talk to Our Team
            </h1>
            <p className="mt-4 text-lg text-fortress-gray-light">
              Have questions about FortressWAF? Want to discuss enterprise
              pricing? We're here to help.
            </p>

            <div className="mt-8 space-y-6">
              <div className="flex items-start gap-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-fortress-navy-light">
                  <Mail className="h-5 w-5 text-fortress-green" />
                </div>
                <div>
                  <h3 className="font-semibold text-white">Email Us</h3>
                  <p className="mt-1 text-fortress-gray-light">
                    security@fortresswaf.io
                  </p>
                  <p className="mt-1 text-sm text-fortress-gray">
                    We respond within 24 hours
                  </p>
                </div>
              </div>

              <div className="flex items-start gap-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-fortress-navy-light">
                  <MessageSquare className="h-5 w-5 text-fortress-green" />
                </div>
                <div>
                  <h3 className="font-semibold text-white">Live Chat</h3>
                  <p className="mt-1 text-fortress-gray-light">
                    Available for Professional and Enterprise customers
                  </p>
                  <p className="mt-1 text-sm text-fortress-gray">
                    Monday - Friday, 9am - 6pm EST
                  </p>
                </div>
              </div>
            </div>

            <div className="mt-12 rounded-2xl bg-fortress-navy-light p-8 border border-fortress-navy-light">
              <h3 className="text-lg font-semibold text-white">
                Enterprise Customers
              </h3>
              <p className="mt-2 text-fortress-gray-light">
                For urgent security incidents or critical issues, our enterprise
                customers have access to 24/7 emergency support.
              </p>
              <p className="mt-4 text-fortress-green font-medium">
                emergency@fortresswaf.io
              </p>
            </div>
          </div>

          <div>
            <Card className="bg-fortress-navy-light border-fortress-navy-light">
              <CardHeader>
                <CardTitle className="text-white">Send us a message</CardTitle>
                <CardDescription className="text-fortress-gray-light">
                  Fill out the form below and we'll get back to you shortly.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                  <div className="space-y-2">
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      placeholder="Your name"
                      {...register("name")}
                      className={
                        errors.name
                          ? "border-red-500 focus-visible:ring-red-500"
                          : ""
                      }
                    />
                    {errors.name && (
                      <p className="text-sm text-red-400">
                        {errors.name.message}
                      </p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="email">Email</Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="you@company.com"
                      {...register("email")}
                      className={
                        errors.email
                          ? "border-red-500 focus-visible:ring-red-500"
                          : ""
                      }
                    />
                    {errors.email && (
                      <p className="text-sm text-red-400">
                        {errors.email.message}
                      </p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="company">Company (optional)</Label>
                    <Input
                      id="company"
                      placeholder="Your company name"
                      {...register("company")}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="message">Message</Label>
                    <Textarea
                      id="message"
                      placeholder="How can we help you?"
                      className={`min-h-[150px] ${
                        errors.message
                          ? "border-red-500 focus-visible:ring-red-500"
                          : ""
                      }`}
                      {...register("message")}
                    />
                    {errors.message && (
                      <p className="text-sm text-red-400">
                        {errors.message.message}
                      </p>
                    )}
                  </div>

                  <Button
                    type="submit"
                    className="w-full bg-fortress-green hover:bg-fortress-green-light"
                    disabled={isSubmitting}
                  >
                    {isSubmitting ? (
                      <>Sending...</>
                    ) : (
                      <>
                        Send Message <Send className="ml-2 h-4 w-4" />
                      </>
                    )}
                  </Button>
                </form>

                <div className="mt-6 text-center text-sm text-fortress-gray">
                  <p>
                    Or email us directly at{" "}
                    <a
                      href="mailto:security@fortresswaf.io"
                      className="text-fortress-green hover:underline"
                    >
                      security@fortresswaf.io
                    </a>
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
