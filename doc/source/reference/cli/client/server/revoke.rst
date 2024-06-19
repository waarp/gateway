==================================
Interdire une règle sur un serveur
==================================

.. program:: waarp-gateway server revoke

Retire au serveur fourni en paramètre la permission d'utiliser la règle donnée.

**Commande**

.. code-block:: shell

   waarp-gateway server revoke "<SERVER>" "<RULE>"

**Options**

**Exemple**

.. code-block:: shell

   waarp-gateway server revoke 'gw_r66' 'règle_1'
