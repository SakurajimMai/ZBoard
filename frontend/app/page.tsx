import Navbar from "@/components/landing/Navbar"
import Hero from "@/components/landing/Hero"
import Features from "@/components/landing/Features"
import Nodes from "@/components/landing/Nodes"
import Pricing from "@/components/landing/Pricing"
import FAQ from "@/components/landing/FAQ"
import Footer from "@/components/landing/Footer"

export default function HomePage() {
  return (
    <main className="min-h-screen bg-background">
      <Navbar />
      <Hero />
      <Features />
      <Nodes />
      <Pricing />
      <FAQ />
      <Footer />
    </main>
  )
}
