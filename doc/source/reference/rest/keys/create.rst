Ajouter une clé cryptographique
===============================

.. http:post:: /api/keys

   Ajoute une nouvelle clé cryptographique avec les informations renseignées en
   format JSON dans le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de la clé cryptographique.
   :reqjson string type: Le type de la clé cryptographique. Les valeurs acceptées
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :reqjson string key: La représentation textuelle de la clé. Si la clé n'est
      pas nativement en format textuel, celle-ci doit être convertie en Base64
      avant son envoi.

   :statuscode 201: La clé a été créée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès à la nouvelle clé créée


   **Exemple de requête**

   .. code-block:: http

      POST https://my_waarp_gateway.net/api/keys HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 103

      {
        "name": "aes-key",
        "type": "AES",
        "privateKey": "0123456789abcdefhijklABCDEFHIJKL"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/keys/aes-key
