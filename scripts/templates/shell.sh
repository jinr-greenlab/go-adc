docker run -it --rm --network host -e TZ={{ tz }} -v {{ config_dir }}:/root/.go-adc -v {{ data_dir }}:/data {{ image }} bash
