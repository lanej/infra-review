import "./style.css";
import { renderReview, type Review } from "./review";

async function load():Promise<Review>{
  const response=await fetch("/api/reviews/pr-1842");
  if(!response.ok) throw new Error(await response.text());
  return response.json();
}

load().then(review=>{document.querySelector<HTMLDivElement>("#app")!.innerHTML=renderReview(review)}).catch(e=>{document.querySelector("#app")!.textContent="Failed to load review: "+e.message});
