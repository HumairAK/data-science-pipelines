# Usage:
# from kfp.dsl.util.yaml_writer import write_pb_to_yaml
# write_pb_to_yaml(pipeline_spec)

yaml_file_path = ("/Users/hukhan/projects/github/rhods/"
                  "data-science-pipelines/sdk/playground/"
                  "pipeline_sample/debug")


def write_pb_to_yaml(message, func):
    from google.protobuf.json_format import MessageToDict
    import yaml
    message_dict = MessageToDict(message, preserving_proto_field_name=True)

    # Convert dictionary to YAML and save to file
    with open(f"{yaml_file_path}/{func.__name__ }.yaml", "w") as yaml_file:
        yaml.dump(message_dict, yaml_file, sort_keys=False)
