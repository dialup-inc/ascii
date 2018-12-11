#include "gst.h"

#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>

GMainLoop *gstreamer_recieve_main_loop = NULL;
void gstreamer_recieve_start_mainloop(void) {
  gstreamer_recieve_main_loop = g_main_loop_new(NULL, FALSE);

  g_main_loop_run(gstreamer_recieve_main_loop);
}

static gboolean gstreamer_recieve_bus_call(GstBus *bus, GstMessage *msg, gpointer data) {
  switch (GST_MESSAGE_TYPE(msg)) {

  case GST_MESSAGE_EOS:
    g_print("End of stream\n");
    exit(1);
    break;

  case GST_MESSAGE_ERROR: {
    gchar *debug;
    GError *error;

    gst_message_parse_error(msg, &error, &debug);
    g_free(debug);

    g_printerr("Error: %s\n", error->message);
    g_error_free(error);
    exit(1);
    break;
  }
  default:
    break;
  }

  return TRUE;
}

GstElement *gstreamer_recieve_create_pipeline(char *pipeline) {
  gst_init(NULL, NULL);
  GError *error = NULL;
  return gst_parse_launch(pipeline, &error);
}

void gstreamer_recieve_start_pipeline(GstElement *pipeline) {
  GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
  gst_bus_add_watch(bus, gstreamer_recieve_bus_call, NULL);
  gst_object_unref(bus);

  gst_element_set_state(pipeline, GST_STATE_PLAYING);
}

void gstreamer_recieve_stop_pipeline(GstElement *pipeline) { gst_element_set_state(pipeline, GST_STATE_NULL); }

void gstreamer_recieve_push_buffer(GstElement *pipeline, void *buffer, int len) {
  GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "src");
  if (src != NULL) {
    gpointer p = g_memdup(buffer, len);
    GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
    gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
  }
}

int gstreamer_receive_pull_sample(GstElement *pipeline, void *result, int len) {
  GstElement *sink = gst_bin_get_by_name(GST_BIN(pipeline), "sink");
  if (sink == NULL) {
    return -1;
  }

  GstSample *sample = gst_app_sink_pull_sample(GST_APP_SINK(sink));
  if (sample == NULL) {
    return -1;
  }

  GstBuffer *buffer = gst_sample_get_buffer(sample);
  if (buffer == NULL) {
    return -1;
  }

  int ret = gst_buffer_extract(buffer, 0, result, len);

  gst_sample_unref(sample);

  return ret;
}
