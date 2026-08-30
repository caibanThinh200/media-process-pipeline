import { NextRequest, NextResponse } from "next/server";

const API_BASE = process.env.API_BASE_URL ?? "";

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Validate required fields
    const { fileName, fileType, mediaType, sizeBytes } = body;
    if (!fileName || !fileType || !mediaType || !sizeBytes) {
      return NextResponse.json(
        { error: "Missing required fields: fileName, fileType, mediaType, sizeBytes" },
        { status: 400 }
      );
    }

    // Forward to the backend upload-api Lambda via API Gateway
    const res = await fetch(`${API_BASE}/upload/presign`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fileName, fileType, mediaType, sizeBytes }),
      cache: "no-store",
    });

    if (!res.ok) {
      const text = await res.text();
      return NextResponse.json(
        { error: `Backend presign failed: ${text}` },
        { status: res.status }
      );
    }

    const data = await res.json();
    // Expect { jobId, uploadUrl, objectKey } from the backend
    return NextResponse.json(data);
  } catch (err) {
    console.error("[presign] unexpected error", err);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
