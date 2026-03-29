import Navbar from "@/components/navbar";
import Hero from "@/components/hero";
import ProvidersStrip from "@/components/providers-strip";
import UseCases from "@/components/use-cases";
import DxShowcase from "@/components/dx-showcase";
import FeaturesGrid from "@/components/features-grid";
import Comparison from "@/components/comparison";
import Architecture from "@/components/architecture";
import GettingStarted from "@/components/getting-started";
import CtaFooter from "@/components/cta-footer";
import Footer from "@/components/footer";

export default function Home() {
  return (
    <div className="min-h-screen flex flex-col bg-dot-grid">
      <Navbar />
      <main>
        <Hero />
        <ProvidersStrip />
        <UseCases />
        <DxShowcase />
        <FeaturesGrid />
        <Comparison />
        <Architecture />
        <GettingStarted />
        <CtaFooter />
      </main>
      <Footer />
    </div>
  );
}
