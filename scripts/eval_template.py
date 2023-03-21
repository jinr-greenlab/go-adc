from jinja2 import Environment, select_autoescape, StrictUndefined, FileSystemLoader
import click
from pathlib import Path
import os

ENV = Environment(
    loader=FileSystemLoader("templates"),
    undefined=StrictUndefined,
    autoescape=select_autoescape(),
)

DISCOVER = True
CONTROL = True
MSTREAM = True
TCPDUMP = True
IMAGE = os.environ.get("IMAGE", "quay.io/kozhukalov/go-adc")
TAG = os.environ.get("TAG", "adc64")
DATA_DIR = Path(os.environ.get("DATA_DIR", os.getcwd()))
CONFIG_DIR = Path(os.environ.get("CONFIG_DIR", os.getcwd()))
TZ = os.environ.get("TZ", "Europe/Moscow")


@click.group()
def cli():
    pass


@cli.command()
@click.option("--config-dir", required=False, default=CONFIG_DIR)
@click.option("--data-dir", default=DATA_DIR)
def shell(config_dir, data_dir):
    template = ENV.get_template("shell.sh")
    rendered = template.render(
        image=f"{IMAGE}:{TAG}",
        config_dir=config_dir,
        data_dir=data_dir,
        tz=TZ,
    )
    print(rendered)


@cli.command()
@click.option("--data-dir", default=DATA_DIR)
@click.option("--config-dir", default=CONFIG_DIR)
@click.option("--tcpdump/--no-tcpdump", default=True, is_flag=True)
@click.option("--discover/--no-discover", default=True, is_flag=True)
@click.option("--control/--no-control", default=True, is_flag=True)
@click.option("--mstream/--no-mstream", default=True, is_flag=True)
def docker_compose(data_dir, config_dir, tcpdump, discover, control, mstream):
    template = ENV.get_template("docker-compose.yaml")
    rendered = template.render(
        image=f"{IMAGE}:{TAG}",
        tcpdump=tcpdump,
        discover=discover,
        control=control,
        mstream=mstream,
        data_dir=data_dir,
        config_dir=config_dir,
        tz=TZ,
    )
    print(rendered)


if __name__ == "__main__":
    cli()
