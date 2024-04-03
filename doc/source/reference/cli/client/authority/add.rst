====================
Ajouter une autorité
====================

.. program:: waarp-gateway authority add

.. describe:: waarp-gateway authority add

Ajoute une nouvelle autorité d'authentification avec les attributs donnés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la nouvelle autorité créée. Doit être unique.

.. option:: -t <TYPE>, --type=<TYPE>

   Le type d'autorité. Actuellement, seuls `tls_authority` et `ssh_cert_authority`
   sont supportés.

.. option:: -i <FILE>, --identity-file=<FILE>

   Le fichier d'identité publique de l'autorité, c'est-à-dire son certificat
   (pour une autorité TLS) ou sa clé publique (pour une autorité SSH).

.. option:: -h <HOST>, --host=<HOST>

   Les hôtes que l'autorité est habilitée à authentifier. Répéter l'option pour
   ajouter plusieurs hôtes. Si vide, l'autorité sera habilité à authentifier
   n'importe quel hôte connu.

|

**Exemple**

Pour ajouter une autorité de certification TLS ayant le droit de certification
pour les hôtes `waarp.fr` et `waarp.org`, la syntaxe est la suivante:

.. code-block:: shell

   waarp-gateway authority add --name 'waarp_ca' --type 'tls_authority' --identity-file './waarp_ca.pem' --host 'waarp.fr' --host 'waarp.org'
