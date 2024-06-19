=====================
Modifier une autorité
=====================

.. program:: waarp-gateway authority update

Modifie les attributs de l'autorité demandée. Les attributs omis restent inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway authority update "<AUTHORITY>"

**Options**

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
   n'importe quel hôte connu. Cette liste remplacera celle déjà existante.
   Pour supprimer tous les hôtes existant, appeler cette options avec un hôte
   vide (`--host ''`).

**Exemple**

Pour changer l'autorité 'waarp_ca' décrite en exemple de la commande `add`, et
lui changer son certificat, ainsi que de changer ses hôtes valides pour que seul
`waarp.org` soit autorisé; la commande est la suivante:

.. code-block:: shell

   waarp-gateway authority update 'waarp_ca' --identity-file ./new_waarp_ca.pem --host 'waarp.org'