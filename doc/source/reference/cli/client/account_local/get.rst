========================
Afficher un compte local
========================

.. program:: waarp-gateway account local get

Affiche les informations du compte donné en paramètre de commande.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" get "<LOGIN>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' get 'tata'
