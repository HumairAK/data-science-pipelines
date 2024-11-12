from pathlib import Path

# Usage:
# from kfp.dsl.util.yaml_writer import write_pb_to_yaml
# write_pb_to_yaml(pipeline_spec)

wd = Path(__file__).parent.parent.parent.parent.parent
yaml_file_path = f"{wd}/playground/pipeline_spec_yamls"


def write_pb_to_yaml(message, func):
    from google.protobuf.json_format import MessageToDict
    import yaml
    message_dict = MessageToDict(message, preserving_proto_field_name=True)

    # Convert dictionary to YAML and save to file
    with open(f"{yaml_file_path}/{func.__name__ }.yaml", "w") as yaml_file:
        yaml.dump(message_dict, yaml_file, sort_keys=False)
