<script lang="ts">
  /** Confidence value 0–100 (percent). */
  export let value: number = 0;

  /** When true, renders an animated shimmer on the fill bar. */
  export let animating: boolean = false;

  $: clamped = Math.min(100, Math.max(0, value));
  $: displayPct = Math.round(clamped);
</script>

<div class="conf-wrap">
  <div class="conf-pct">{displayPct}%</div>
  <div class="conf-bar">
    <div
      class="conf-fill"
      class:shimmer={animating}
      style="width: {clamped}%"
    ></div>
  </div>
  <div class="conf-label">Confidence</div>
</div>

<style>
  .conf-wrap {
    text-align: right;
  }

  .conf-pct {
    font-family: "Space Grotesk", sans-serif;
    font-weight: 700;
    font-size: 1.125rem;
    color: var(--ink-900);
    line-height: 1;
  }

  .conf-bar {
    height: 6px;
    background: #e7eef9;
    border-radius: 999px;
    width: 140px;
    overflow: hidden;
    margin-top: 5px;
  }

  .conf-fill {
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
    transition: width 0.6s ease;
    position: relative;
  }

  .conf-fill.shimmer::after {
    content: "";
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(255, 255, 255, 0.45) 50%,
      transparent 100%
    );
    background-size: 200% 100%;
    animation: shimmer-anim 1.4s linear infinite;
  }

  @keyframes shimmer-anim {
    0% {
      background-position: -200% 0;
    }
    100% {
      background-position: 200% 0;
    }
  }

  .conf-label {
    font-size: 0.6875rem;
    color: var(--ink-500);
    margin-top: 3px;
  }
</style>
