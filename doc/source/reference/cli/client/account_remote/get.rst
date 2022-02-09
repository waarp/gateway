==========================
Afficher un compte distant
==========================

.. program:: waarp-gateway account remote get

.. describe:: waarp-gateway account remote <PARTNER> get <LOGIN>

Affiche les informations du compte demandé en paramètre de commande.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a http://user:password@remotehost:8080 account remote 'waarp_sftp' get 'titi'