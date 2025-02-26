Consulter une clé cryptographique
=================================

.. http:get:: /api/keys/(string:key_name)

   Renvoie la clé cryptographique demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: La clé a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La clé demandée n'existe pas

   :resjson string name: Le nom de la clé cryptographique.
   :resjson string type: Le type de la clé cryptographique. Les valeurs possibles
      sont :

      - ``AES`` pour les clés de (dé)chiffrement AES
      - ``HMAC`` pour les clés de signature HMAC
      - ``PGP-PUBLIC`` pour les clés PGP publiques
      - ``PGP-PRIVATE`` pour les clés PGP privées
   :resjson string key: La représentation textuelle de la clé. Si la clé n'est
      pas nativement en format textuel, celle-ci sera convertie en Base64.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/keys/aes-key HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 98

      {
        "name": "aes-key",
        "type": "AES",
        "privateKey": "0123456789abcdefhijklABCDEFHIJKL"
      }
