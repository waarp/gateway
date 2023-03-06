###################
Status des services
###################

.. program:: waarp-gateway status

.. describe:: waarp-gateway status

Affiche le status des différents services de la gateway. Cela inclus les services
nécessaires au fonctionnement de la gateway, ainsi que tous les serveurs locaux
et des clients rattachés à celle-ci.

La commande affiche également la version de la gateway et la date locale.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' status