###################
Status des services
###################

.. program:: waarp-gateway status

Affiche le status des différents services de la gateway. Cela inclus les services
nécessaires au fonctionnement de la gateway, ainsi que tous les serveurs locaux
et des clients rattachés à celle-ci.

La commande affiche également la version du service Gateway ainsi que son heure
locale.

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway status
